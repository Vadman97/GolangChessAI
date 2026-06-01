package analysis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type UCIReplayState struct {
	Ply            int
	Move           string
	Side           color.Color
	Status         byte
	Repeats        int
	CumulativeReps int
	HalfmoveClock  int
	FENAfter       string
	LegalMoveCount int
}

func RunUCIReplay(moveText string, verbose bool) error {
	states, err := ReplayUCIMoves(moveText)
	if err != nil {
		return err
	}
	for _, state := range states {
		fmt.Printf("ply=%d side=%s move=%s status=%s repeats=%d cumulative=%d halfmove=%d legal=%d\n",
			state.Ply,
			color.Names[state.Side],
			state.Move,
			game.StatusStrings[state.Status],
			state.Repeats,
			state.CumulativeReps,
			state.HalfmoveClock,
			state.LegalMoveCount,
		)
		if verbose {
			fmt.Printf("  fen=%s\n", state.FENAfter)
		}
	}
	if len(states) > 0 {
		last := states[len(states)-1]
		fmt.Printf("Final: status=%s repeats=%d cumulative=%d fen=%s\n",
			game.StatusStrings[last.Status], last.Repeats, last.CumulativeReps, last.FENAfter)
	}
	return nil
}

func RunLichessStateReplay(logPath, gameID string, verbose bool) error {
	moves, replayGameID, err := ExtractLichessMovesFromLog(logPath, gameID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(moves) == "" {
		return fmt.Errorf("no moves found for game %q in %s", gameID, logPath)
	}
	fmt.Printf("Replaying %d plies from game %s\n", len(strings.Fields(moves)), replayGameID)
	return RunUCIReplay(moves, verbose)
}

func ReplayUCIMoves(moveText string) ([]UCIReplayState, error) {
	tokens := strings.Fields(moveText)
	b := &board.Board{}
	b.ResetDefault()
	side := color.White
	fullMove := 1
	var previousMove *board.LastMove
	states := make([]UCIReplayState, 0, len(tokens))

	for i, token := range tokens {
		m, err := matchUCIMove(b, side, previousMove, token)
		if err != nil {
			return nil, fmt.Errorf("ply %d %s: %w", i+1, token, err)
		}
		previousMove = board.MakeMove(&m, b)
		nextSide := side ^ 1
		if side == color.Black {
			fullMove++
		}
		status := statusAfterMove(b, nextSide, previousMove)
		legalMoves := b.GetAllMoves(nextSide, previousMove)
		states = append(states, UCIReplayState{
			Ply:            i + 1,
			Move:           token,
			Side:           side,
			Status:         status,
			Repeats:        b.CurrentPositionRepeats,
			CumulativeReps: b.PreviousPositionsSeen,
			HalfmoveClock:  b.MovesSinceNoDraw,
			FENAfter:       BoardToFEN(b, nextSide, previousMove, fullMove),
			LegalMoveCount: len(*legalMoves),
		})
		side = nextSide
	}
	return states, nil
}

func ExtractLichessMovesFromLog(logPath, gameID string) (string, string, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	reGameFull := regexp.MustCompile(`\\"id\\":\\"([^"]+)\\"|\"id\":\"([^"]+)\"`)
	reMoves := regexp.MustCompile(`\\"moves\\":\\"([^"]*)\\"|\"moves\":\"([^"]*)"`)
	active := gameID == ""
	var latest string
	var latestGameID string
	var activeGameID string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `\"type\":\"gameFull\"`) || strings.Contains(line, `"type":"gameFull"`) {
			if m := reGameFull.FindStringSubmatch(line); m != nil {
				id := firstNonEmpty(m[1], m[2])
				active = gameID == "" || id == gameID
				if active {
					activeGameID = id
				}
			}
		}
		if !active {
			continue
		}
		if m := reMoves.FindStringSubmatch(line); m != nil {
			latest = firstNonEmpty(m[1], m[2])
			latestGameID = activeGameID
		}
	}
	if latestGameID == "" {
		latestGameID = gameID
	}
	return latest, latestGameID, scanner.Err()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func matchUCIMove(b *board.Board, side color.Color, previousMove *board.LastMove, uci string) (location.Move, error) {
	target, promoType, err := parseUCIMoveText(uci)
	if err != nil {
		return location.Move{}, err
	}
	moves := b.GetAllMoves(side, previousMove)
	for _, m := range *moves {
		if !m.Start.Equals(target.Start) {
			continue
		}
		if m.End.GetRow() != target.End.GetRow() || m.End.GetCol() != target.End.GetCol() {
			continue
		}
		hasPromo, actualPromo := m.End.GetPawnPromotion()
		if promoType != piece.NilType {
			if !hasPromo || actualPromo != promoType {
				continue
			}
		} else if hasPromo {
			continue
		}
		return m, nil
	}
	return location.Move{}, fmt.Errorf("move is not legal")
}

func parseUCIMoveText(uci string) (location.Move, byte, error) {
	if len(uci) != 4 && len(uci) != 5 {
		return location.Move{}, piece.NilType, fmt.Errorf("invalid UCI length")
	}
	start, err := parseUCISquare(uci[:2])
	if err != nil {
		return location.Move{}, piece.NilType, err
	}
	end, err := parseUCISquare(uci[2:4])
	if err != nil {
		return location.Move{}, piece.NilType, err
	}
	promoType := piece.NilType
	if len(uci) == 5 {
		switch uci[4] {
		case 'q':
			promoType = piece.QueenType
		case 'r':
			promoType = piece.RookType
		case 'b':
			promoType = piece.BishopType
		case 'n':
			promoType = piece.KnightType
		default:
			return location.Move{}, piece.NilType, fmt.Errorf("invalid promotion piece %q", uci[4])
		}
	}
	return location.Move{Start: start, End: end}, promoType, nil
}

func parseUCISquare(square string) (location.Location, error) {
	if len(square) != 2 || square[0] < 'a' || square[0] > 'h' || square[1] < '1' || square[1] > '8' {
		return location.Location{}, fmt.Errorf("invalid square %q", square)
	}
	col := location.CoordinateType(7 - (square[0] - 'a'))
	row := location.CoordinateType(square[1] - '1')
	return location.NewLocation(row, col), nil
}

func statusAfterMove(b *board.Board, nextSide color.Color, previousMove *board.LastMove) byte {
	if b.IsInCheckmate(nextSide, previousMove) {
		if nextSide == color.White {
			return game.BlackWin
		}
		return game.WhiteWin
	}
	if b.IsStalemate(nextSide, previousMove) {
		return game.Stalemate
	}
	if b.MovesSinceNoDraw >= 100 {
		return game.FiftyMoveDraw
	}
	if b.CurrentPositionRepeats >= 2 {
		return game.RepeatedActionThreeTimeDraw
	}
	if b.IsInsufficientMaterial() {
		return game.InsufficientMaterialDraw
	}
	return game.Active
}

func PrintUCIReplayJSON(states []UCIReplayState) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(states)
}
