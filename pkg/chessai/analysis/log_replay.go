package analysis

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

var (
	reBestMove   = regexp.MustCompile(`Best D:\d+ M:move from \((\d+), (\d+)\) to \((\d+), (\d+)\)`)
	reMoveHeader = regexp.MustCompile(`Move #(\d+) by (White|Black)`)
	reAIScore    = regexp.MustCompile(`best move leads to score (-?\d+)`)
	reBoardRow   = regexp.MustCompile(`^(\d) (.+) \d$`)
	reAIColor    = regexp.MustCompile(`Player AI .* - (White|Black)\) thinking`)
	reGameFullID = regexp.MustCompile(`\\"id\\":\\"([^"]+)\\"|\"id\":\"([^"]+)\"`)
)

type logEntry struct {
	gameIndex int
	plyNum    int
	clr       color.Color
	fromRow   uint8
	fromCol   uint8
	toRow     uint8
	toCol     uint8
	aiScore   int
	hasScore  bool
	grid      [board.Height]string // raw "|"-separated piece strings per row
}

// parseLichessLog reads the internal game log and extracts one entry per AI move.
// Each entry includes the move coordinates (from the last "Best D:" line) and the
// board state shown after the move.
func parseLichessLog(path string) ([]logEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []logEntry
	var (
		hasLastMove              bool
		lastFromRow, lastFromCol uint8
		lastToRow, lastToCol     uint8
		pendingScore             int
		hasPendingScore          bool
		// state for reading board grid after a move header
		readingBoard bool
		pendingEntry logEntry
		gameIndex    int
	)

	reGameOver := regexp.MustCompile(`Game Over!|Game state:.*(?:Win|Draw|Resign)`)

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()

		// Board row: "0 W_R|...|W_R 0"
		if readingBoard {
			if m := reBoardRow.FindStringSubmatch(line); m != nil {
				r, _ := strconv.Atoi(m[1])
				pendingEntry.grid[r] = m[2]
				if r == board.Height-1 {
					// last row collected — commit the entry
					entries = append(entries, pendingEntry)
					readingBoard = false
				}
				continue
			}
			// Skip header/footer lines (e.g. "   0   1   2   3   4   5   6   7  ")
			// and empty lines — they're part of the board display block.
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || (len(trimmed) > 0 && trimmed[0] >= '0' && trimmed[0] <= '9' && strings.Contains(trimmed, " ")) {
				continue
			}
			// Any other non-board content cancels board reading
			readingBoard = false
		}

		// Game boundary: reset move state so the last move from one game
		// doesn't contaminate the first (book-move) entry of the next game.
		if reGameOver.MatchString(line) {
			hasLastMove = false
			hasPendingScore = false
			gameIndex++
			continue
		}

		if m := reBestMove.FindStringSubmatch(line); m != nil {
			r1, _ := strconv.Atoi(m[1])
			c1, _ := strconv.Atoi(m[2])
			r2, _ := strconv.Atoi(m[3])
			c2, _ := strconv.Atoi(m[4])
			lastFromRow, lastFromCol = uint8(r1), uint8(c1)
			lastToRow, lastToCol = uint8(r2), uint8(c2)
			hasLastMove = true
			continue
		}

		if m := reAIScore.FindStringSubmatch(line); m != nil {
			pendingScore, _ = strconv.Atoi(m[1])
			hasPendingScore = true
			continue
		}

		if m := reMoveHeader.FindStringSubmatch(line); m != nil {
			ply, _ := strconv.Atoi(m[1])
			clr := color.White
			if m[2] == "Black" {
				clr = color.Black
			}
			// Skip entries where the last Best D move was a null/zero move
			// (engine fell back to random; we don't know the actual move played).
			isNullMove := lastFromRow == 0 && lastFromCol == 0 && lastToRow == 0 && lastToCol == 0
			if hasLastMove && !isNullMove {
				pendingEntry = logEntry{
					gameIndex: gameIndex,
					plyNum:    ply,
					clr:       clr,
					fromRow:   lastFromRow,
					fromCol:   lastFromCol,
					toRow:     lastToRow,
					toCol:     lastToCol,
				}
				if hasPendingScore {
					pendingEntry.aiScore = pendingScore
					pendingEntry.hasScore = true
				}
				readingBoard = true
			}
			hasLastMove = false
			hasPendingScore = false
		}
	}
	return entries, scanner.Err()
}

// boardFromGrid constructs a Board from a parsed 8-row grid and infers castling flags
// from whether the king and rooks are still on their starting squares.
func boardFromGrid(grid [board.Height]string) *board.Board {
	b := &board.Board{}
	for r := 0; r < board.Height; r++ {
		cells := strings.Split(grid[r], "|")
		for c, cell := range cells {
			l := location.NewLocation(location.CoordinateType(r), location.CoordinateType(c))
			if cell == "   " || len(cell) != 3 {
				b.SetPiece(l, nil)
				continue
			}
			parts := strings.Split(cell, "_")
			if len(parts) != 2 {
				b.SetPiece(l, nil)
				continue
			}
			cChar, pChar := rune(parts[0][0]), rune(parts[1][0])
			pType := pieceNameToType(pChar)
			if pType == piece.NilType {
				b.SetPiece(l, nil)
				continue
			}
			p := board.PieceFromType(pType)
			p.SetColor(board.ColorFromChar(cChar))
			p.SetPosition(l)
			b.SetPiece(l, p)
			if pType == piece.KingType {
				b.KingLocations[p.GetColor()] = l
			}
		}
	}

	// Infer castling: if king/rooks are on starting squares, assume right still exists.
	// This is heuristic and breaks if a piece moved away and returned, but is correct
	// for the vast majority of game positions.
	for _, c := range []color.Color{color.White, color.Black} {
		pieceRow := board.StartRow[c]["Piece"]
		kingCol := location.CoordinateType(3) // engine col 3 = standard e-file

		if p := b.GetPiece(location.NewLocation(pieceRow, kingCol)); p == nil || p.GetPieceType() != piece.KingType || p.GetColor() != c {
			b.SetFlag(board.FlagKingMoved, c, true)
		}
		// Left rook (col 0 = h-file = kingside)
		if p := b.GetPiece(location.NewLocation(pieceRow, 0)); p == nil || p.GetPieceType() != piece.RookType || p.GetColor() != c {
			b.SetFlag(board.FlagLeftRookMoved, c, true)
		}
		// Right rook (col 7 = a-file = queenside)
		if p := b.GetPiece(location.NewLocation(pieceRow, 7)); p == nil || p.GetPieceType() != piece.RookType || p.GetColor() != c {
			b.SetFlag(board.FlagRightRookMoved, c, true)
		}
	}

	return b
}

// pieceFromGrid returns the piece type+color string at a given cell in a grid row,
// or "" if empty.
func pieceFromGrid(grid [board.Height]string, row, col int) string {
	if row < 0 || row >= board.Height {
		return ""
	}
	cells := strings.Split(grid[row], "|")
	if col < 0 || col >= len(cells) {
		return ""
	}
	s := cells[col]
	if s == "   " {
		return ""
	}
	return s
}

// unApplyMove reconstructs the position BEFORE the given move was made (W_n)
// from the post-move board (B_n) and the previous board (B_prev, may be nil).
// It handles: simple moves, captures (via B_prev), en passant, and castling.
func unApplyMove(bn *board.Board, fromRow, fromCol, toRow, toCol uint8,
	movedColor color.Color, bnGrid, bPrevGrid *[board.Height]string) *board.Board {

	// Clone B_n by re-building a board from its current grid
	wn := boardFromGrid(*bnGrid)

	fromLoc := location.NewLocation(fromRow, fromCol)
	toLoc := location.NewLocation(toRow, toCol)

	movedPiece := wn.GetPiece(toLoc)
	if movedPiece == nil {
		return wn // can't un-apply; something is wrong
	}

	// Check for promotion: if the piece at toLoc is a Queen but in a pawn-promotion row,
	// restore a Pawn at fromLoc instead.
	isPromotion := false
	promotionRow := board.StartRow[movedColor^1]["Piece"]
	if toRow == promotionRow && movedPiece.GetPieceType() == piece.QueenType {
		// Heuristic: assume it was a pawn that promoted
		isPromotion = true
	}

	// Clear the destination
	wn.SetPiece(toLoc, nil)

	// Restore the moved piece (or pawn before promotion) at origin
	var restored board.Piece
	if isPromotion {
		p := board.PieceFromType(piece.PawnType)
		p.SetColor(movedColor)
		p.SetPosition(fromLoc)
		restored = p
	} else {
		// No SetPosition: decoded pieces are shared immutable instances, and
		// SetPiece encodes only type+color (position is implicit in the square).
		restored = movedPiece
	}
	wn.SetPiece(fromLoc, restored)

	// Update king location if necessary
	if restored.GetPieceType() == piece.KingType {
		wn.KingLocations[movedColor] = fromLoc
	}

	// Handle castling: king moved 2 squares → also restore rook
	if restored.GetPieceType() == piece.KingType {
		colDiff := int(toCol) - int(fromCol)
		if colDiff == 2 {
			// King moved left (toward col 0, h-file) = kingside castle
			// Rook moved from col 0 to col 2
			rookFrom := location.NewLocation(fromRow, 0)
			rookTo := location.NewLocation(fromRow, 2)
			rook := wn.GetPiece(rookTo)
			if rook != nil {
				wn.SetPiece(rookTo, nil)
				wn.SetPiece(rookFrom, rook)
			}
		} else if colDiff == -2 {
			// King moved right (toward col 7, a-file) = queenside castle
			// Rook moved from col 7 to col 4
			rookFrom := location.NewLocation(fromRow, 7)
			rookTo := location.NewLocation(fromRow, 4)
			rook := wn.GetPiece(rookTo)
			if rook != nil {
				wn.SetPiece(rookTo, nil)
				wn.SetPiece(rookFrom, rook)
			}
		}
	}

	// Handle en passant: pawn moved diagonally to empty square
	// The captured pawn is at (fromRow, toCol)
	if restored.GetPieceType() == piece.PawnType && fromCol != toCol {
		capturedLoc := location.NewLocation(fromRow, toCol)
		if wn.GetPiece(capturedLoc) == nil {
			// Restore the en-passant-captured pawn
			captPawn := board.PieceFromType(piece.PawnType)
			captPawn.SetColor(movedColor ^ 1)
			captPawn.SetPosition(capturedLoc)
			wn.SetPiece(capturedLoc, captPawn)
		}
	}

	// Restore captured piece (if any) from the previous Black board state
	// B_prev shows position after previous Black move; if B_prev[toRow][toCol] had an
	// opponent piece, it was likely captured (unless White moved it there between B_prev and W_n).
	if bPrevGrid != nil && !isPromotion {
		prevCell := pieceFromGrid(*bPrevGrid, int(toRow), int(toCol))
		if prevCell != "" && len(prevCell) == 3 {
			// Check that it belongs to the opponent of the mover
			if (prevCell[0] == 'W' && movedColor == color.Black) || (prevCell[0] == 'B' && movedColor == color.White) {
				// Restore the captured piece
				d := strings.Split(prevCell, "_")
				cChar, pChar := rune(d[0][0]), rune(d[1][0])
				pType := pieceNameToType(pChar)
				if pType != piece.NilType {
					capPiece := board.PieceFromType(pType)
					capPiece.SetColor(board.ColorFromChar(cChar))
					capPiece.SetPosition(toLoc)
					wn.SetPiece(toLoc, capPiece)
				}
			}
		}
	}

	// Re-infer castling rights for W_n
	for _, c := range []color.Color{color.White, color.Black} {
		pieceRow := board.StartRow[c]["Piece"]
		kingCol := location.CoordinateType(3)
		if p := wn.GetPiece(location.NewLocation(pieceRow, kingCol)); p == nil || p.GetPieceType() != piece.KingType || p.GetColor() != c {
			wn.SetFlag(board.FlagKingMoved, c, true)
		}
		if p := wn.GetPiece(location.NewLocation(pieceRow, 0)); p == nil || p.GetPieceType() != piece.RookType || p.GetColor() != c {
			wn.SetFlag(board.FlagLeftRookMoved, c, true)
		}
		if p := wn.GetPiece(location.NewLocation(pieceRow, 7)); p == nil || p.GetPieceType() != piece.RookType || p.GetColor() != c {
			wn.SetFlag(board.FlagRightRookMoved, c, true)
		}
	}

	return wn
}

func pieceNameToType(ch rune) byte {
	switch ch {
	case 'R':
		return piece.RookType
	case 'N':
		return piece.KnightType
	case 'B':
		return piece.BishopType
	case 'Q':
		return piece.QueenType
	case 'K':
		return piece.KingType
	case 'P':
		return piece.PawnType
	}
	return piece.NilType
}

const (
	mistakeThreshold    = 50
	inaccuracyThreshold = 20
)

func classifyLoss(cpLoss int, played, sfBest string) string {
	if played == sfBest {
		return ""
	}
	switch {
	case cpLoss >= BlunderThreshold:
		return "BLUNDER"
	case cpLoss >= mistakeThreshold:
		return "MISTAKE"
	case cpLoss >= inaccuracyThreshold:
		return "INACCURACY"
	default:
		return ""
	}
}

func colorName(c color.Color) string {
	if c == color.White {
		return "White"
	}
	return "Black"
}

type replayEntry struct {
	logEntry
	before      *board.Board
	after       *board.Board
	prevMove    *board.LastMove
	afterMove   *board.LastMove
	move        location.Move
	engineUCI   string
	fullMoveNum int
}

func boardsEqual(a, b *board.Board) bool {
	for row := location.CoordinateType(0); row < board.Height; row++ {
		for col := location.CoordinateType(0); col < board.Width; col++ {
			l := location.NewLocation(row, col)
			ap := a.GetPiece(l)
			bp := b.GetPiece(l)
			if ap == nil || bp == nil {
				if ap != bp {
					return false
				}
				continue
			}
			if ap.GetColor() != bp.GetColor() || ap.GetPieceType() != bp.GetPieceType() {
				return false
			}
		}
	}
	return true
}

func sameMoveSquare(a, b location.Move) bool {
	return a.Start.Equals(b.Start) && a.End.GetRow() == b.End.GetRow() && a.End.GetCol() == b.End.GetCol()
}

func matchingLegalMove(b *board.Board, c color.Color, previousMove *board.LastMove, target location.Move) (location.Move, bool) {
	moves := b.GetAllMoves(c, previousMove)
	for _, m := range *moves {
		if sameMoveSquare(m, target) {
			return m, true
		}
	}
	return location.Move{}, false
}

func replayLogEntries(entries []logEntry) ([]replayEntry, error) {
	var replayed []replayEntry
	state := &board.Board{}
	state.ResetDefault()
	sideToMove := color.White
	var previousMove *board.LastMove
	currentGame := -1

	for _, entry := range entries {
		if entry.gameIndex != currentGame {
			state = &board.Board{}
			state.ResetDefault()
			sideToMove = color.White
			previousMove = nil
			currentGame = entry.gameIndex
		}

		targetAfter := boardFromGrid(entry.grid)
		knownMove := location.Move{
			Start: location.NewLocation(entry.fromRow, entry.fromCol),
			End:   location.NewLocation(entry.toRow, entry.toCol),
		}
		if sideToMove != entry.clr {
			var inferredBefore *board.Board
			var inferredPrev *board.LastMove
			ok := false
			moves := state.GetAllMoves(sideToMove, previousMove)
			for _, opponentMove := range *moves {
				candidateBefore := state.Copy()
				opponentLast := board.MakeMove(&opponentMove, candidateBefore)
				legalAI, legalOK := matchingLegalMove(candidateBefore, entry.clr, opponentLast, knownMove)
				if !legalOK {
					continue
				}
				candidateAfter := candidateBefore.Copy()
				board.MakeMove(&legalAI, candidateAfter)
				if boardsEqual(candidateAfter, targetAfter) {
					inferredBefore = candidateBefore
					inferredPrev = opponentLast
					ok = true
					break
				}
			}
			if !ok {
				return nil, fmt.Errorf("could not infer opponent move before ply %d logged move %s side %s", entry.plyNum, knownMove, colorName(entry.clr))
			}
			state = inferredBefore
			previousMove = inferredPrev
			sideToMove = entry.clr
		}

		legalMove, ok := matchingLegalMove(state, entry.clr, previousMove, knownMove)
		if !ok {
			return nil, fmt.Errorf("logged move %s is not legal at ply %d", knownMove, entry.plyNum)
		}
		before := state.Copy()
		prevForFEN := previousMove
		after := state.Copy()
		afterMove := board.MakeMove(&legalMove, after)
		previousMove = afterMove
		if !boardsEqual(after, targetAfter) {
			return nil, fmt.Errorf("logged board does not match replay after ply %d", entry.plyNum)
		}
		replayed = append(replayed, replayEntry{
			logEntry:    entry,
			before:      before,
			after:       after,
			prevMove:    prevForFEN,
			afterMove:   afterMove,
			move:        legalMove,
			engineUCI:   MoveToUCI(legalMove),
			fullMoveNum: (entry.plyNum + 1) / 2,
		})
		state = after
		sideToMove = entry.clr ^ 1
	}
	return replayed, nil
}

type LogReplayConfig struct {
	LogPath        string
	StockfishPath  string
	StockfishDepth int
	AppendFENsPath string
	AppendMinLoss  int
}

type replayFinding struct {
	FEN      string
	Tag      string
	Expected string
	Bad      string
	Notes    string
}

// RunLogReplay parses the internal lichess game log, replays each AI move through
// local Stockfish, and prints a blunder report. The log typically contains only
// one side's moves (the side the AI is playing); White moves from the opponent
// are not logged.
//
// Usage: ./main log-replay [logPath] [sfDepth] [stockfishPath]
func RunLogReplay(logPath, stockfishPath string, sfDepth int) {
	RunLogReplayWithConfig(LogReplayConfig{
		LogPath:        logPath,
		StockfishPath:  stockfishPath,
		StockfishDepth: sfDepth,
	})
}

func RunLogReplayWithConfig(cfg LogReplayConfig) {
	logPath := cfg.LogPath
	stockfishPath := cfg.StockfishPath
	sfDepth := cfg.StockfishDepth
	if logPath == "" {
		logPath = "/tmp/chess.lichess.log"
	}
	if stockfishPath == "" {
		stockfishPath = "./stockfish"
	}
	if sfDepth <= 0 {
		sfDepth = 15
	}
	appendMinLoss := cfg.AppendMinLoss
	if appendMinLoss <= 0 {
		appendMinLoss = mistakeThreshold
	}
	entries, err := parseLichessLog(logPath)
	if err != nil {
		log.Fatalf("Failed to parse log %s: %v", logPath, err)
	}
	if len(entries) == 0 {
		fmt.Println("No moves found in log.")
		return
	}
	replayed, err := replayLogEntries(entries)
	if err != nil {
		fmt.Printf("Board-grid replay failed (%v); falling back to Lichess gameState moves.\n\n", err)
		runLichessMoveTextStockfishReplay(LogReplayConfig{
			LogPath:        logPath,
			StockfishPath:  stockfishPath,
			StockfishDepth: sfDepth,
			AppendFENsPath: cfg.AppendFENsPath,
			AppendMinLoss:  appendMinLoss,
		})
		return
	}

	fmt.Printf("Parsed %d moves from %s\n\n", len(entries), logPath)

	sf, err := NewStockfishEngine(stockfishPath)
	if err != nil {
		log.Fatalf("Cannot start Stockfish: %v", err)
	}
	defer sf.Close()

	totalBlunders, totalMistakes, totalInaccuracies := 0, 0, 0
	findings := []replayFinding{}

	for _, replay := range replayed {
		entry := replay.logEntry
		// Use the color recorded in each entry — the log may contain games where
		// the AI plays White in one game and Black in another.
		entryColor := entry.clr

		// FEN before AI's move: AI's color is to move
		fenBefore := BoardToFEN(replay.before, entryColor, replay.prevMove, replay.fullMoveNum)
		sfBefore := sf.Analyze(fenBefore, sfDepth)

		// FEN after AI's move: opponent's turn
		opponent := entryColor ^ 1
		fenAfter := BoardToFEN(replay.after, opponent, replay.afterMove, replay.fullMoveNum)
		sfAfter := sf.Analyze(fenAfter, sfDepth)

		// centipawn loss for the side that moved (AI)
		cpSTMBefore := sfBefore.CentipawnsSTM
		cpSTMAfterOpponent := sfAfter.CentipawnsSTM
		cpLoss := cpSTMBefore + cpSTMAfterOpponent
		if cpLoss < 0 || replay.engineUCI == sfBefore.BestMove {
			// Clamp negatives (noise) and moves that match SF's recommendation
			// (apparent loss is a depth-consistency artifact, not a real error).
			cpLoss = 0
		}
		// Cap at 2000 cp so mate-score arithmetic doesn't produce misleading large values.
		if cpLoss > 2000 {
			cpLoss = 2000
		}

		label := classifyLoss(cpLoss, replay.engineUCI, sfBefore.BestMove)
		switch label {
		case "BLUNDER":
			totalBlunders++
		case "MISTAKE":
			totalMistakes++
		case "INACCURACY":
			totalInaccuracies++
		}

		// Format move number for display
		moveLabel := fmt.Sprintf("%d.", replay.fullMoveNum)
		if entryColor == color.Black {
			moveLabel = fmt.Sprintf("%d...", replay.fullMoveNum)
		}

		sfEvalStr := ""
		if sfBefore.IsMate {
			if sfBefore.MateIn > 0 {
				sfEvalStr = fmt.Sprintf("M+%d", sfBefore.MateIn)
			} else {
				sfEvalStr = fmt.Sprintf("M%d", sfBefore.MateIn)
			}
		} else {
			sfEvalStr = fmt.Sprintf("%+d cp", cpSTMBefore)
		}

		aiEvalStr := ""
		if entry.hasScore {
			aiEvalStr = fmt.Sprintf("  AI=%+d", entry.aiScore)
		}

		flagStr := ""
		if label != "" {
			flagStr = fmt.Sprintf("  [%s]", label)
		}

		fmt.Printf("Ply %2d  %s%s  SF: %s  loss: %+d  SF-best: %s%s%s\n",
			entry.plyNum, moveLabel, replay.engineUCI, sfEvalStr, cpLoss, sfBefore.BestMove, aiEvalStr, flagStr)

		if label == "BLUNDER" || label == "MISTAKE" {
			fmt.Printf("  AI played %-6s but SF recommends %-6s  (diff: %+d cp)\n", replay.engineUCI, sfBefore.BestMove, cpLoss)
			fmt.Printf("  FEN: %s\n\n", fenBefore)
			if cpLoss >= appendMinLoss {
				findings = append(findings, replayFinding{
					FEN:      fenBefore,
					Tag:      fmt.Sprintf("lichess-log-g%d-ply%d", entry.gameIndex+1, entry.plyNum),
					Expected: sfBefore.BestMove,
					Bad:      replay.engineUCI,
					Notes:    fmt.Sprintf("%s loss=%dcp sf-depth=%d", strings.ToLower(label), cpLoss, sfDepth),
				})
			}
		}
	}

	total := len(replayed)
	fmt.Printf("\n=== Summary: %d AI half-moves ===\n", total)
	fmt.Printf("  Blunders     (>=%d cp): %d\n", BlunderThreshold, totalBlunders)
	fmt.Printf("  Mistakes     (>=%d cp): %d\n", mistakeThreshold, totalMistakes)
	fmt.Printf("  Inaccuracies (>=%d cp): %d\n", inaccuracyThreshold, totalInaccuracies)
	if cfg.AppendFENsPath != "" {
		if err := appendReplayFindings(cfg.AppendFENsPath, findings); err != nil {
			log.Fatalf("Failed to append FENs to %s: %v", cfg.AppendFENsPath, err)
		}
	}
}

type uciAnalysisEntry struct {
	ply         int
	fullMoveNum int
	side        color.Color
	move        location.Move
	uci         string
	fenBefore   string
	fenAfter    string
}

func runLichessMoveTextStockfishReplay(cfg LogReplayConfig) {
	logPath := cfg.LogPath
	stockfishPath := cfg.StockfishPath
	sfDepth := cfg.StockfishDepth
	appendMinLoss := cfg.AppendMinLoss
	if appendMinLoss <= 0 {
		appendMinLoss = mistakeThreshold
	}
	moves, gameID, err := ExtractLichessMovesFromLog(logPath, "")
	if err != nil {
		log.Fatalf("Failed to extract gameState moves from %s: %v", logPath, err)
	}
	if strings.TrimSpace(moves) == "" {
		log.Fatalf("No gameState moves found in %s", logPath)
	}
	botColor, ok := inferBotColorFromLog(logPath, gameID)
	if !ok {
		log.Fatalf("Could not infer bot color from %s", logPath)
	}
	entries, err := replayUCIAnalysisEntries(moves)
	if err != nil {
		log.Fatalf("Failed to replay gameState moves from %s: %v", logPath, err)
	}

	sf, err := NewStockfishEngine(stockfishPath)
	if err != nil {
		log.Fatalf("Cannot start Stockfish: %v", err)
	}
	defer sf.Close()

	fmt.Printf("Parsed %d plies from game %s; analyzing %s bot moves from %s\n\n",
		len(strings.Fields(moves)), gameID, colorName(botColor), logPath)

	totalBotMoves, totalBlunders, totalMistakes, totalInaccuracies := 0, 0, 0, 0
	findings := []replayFinding{}
	for _, entry := range entries {
		if entry.side != botColor {
			continue
		}
		totalBotMoves++
		sfBefore := sf.Analyze(entry.fenBefore, sfDepth)
		sfAfter := sf.Analyze(entry.fenAfter, sfDepth)
		cpLoss := sfBefore.CentipawnsSTM + sfAfter.CentipawnsSTM
		if cpLoss < 0 || entry.uci == sfBefore.BestMove {
			cpLoss = 0
		}
		if cpLoss > 2000 {
			cpLoss = 2000
		}
		label := classifyLoss(cpLoss, entry.uci, sfBefore.BestMove)
		switch label {
		case "BLUNDER":
			totalBlunders++
		case "MISTAKE":
			totalMistakes++
		case "INACCURACY":
			totalInaccuracies++
		}
		moveLabel := fmt.Sprintf("%d.", entry.fullMoveNum)
		if entry.side == color.Black {
			moveLabel = fmt.Sprintf("%d...", entry.fullMoveNum)
		}
		flagStr := ""
		if label != "" {
			flagStr = fmt.Sprintf("  [%s]", label)
		}
		fmt.Printf("Ply %3d  %s%s  SF: %+d cp  loss: %+d  SF-best: %s%s\n",
			entry.ply, moveLabel, entry.uci, sfBefore.CentipawnsSTM, cpLoss, sfBefore.BestMove, flagStr)
		if label == "BLUNDER" || label == "MISTAKE" {
			fmt.Printf("  AI played %-6s but SF recommends %-6s  (diff: %+d cp)\n", entry.uci, sfBefore.BestMove, cpLoss)
			fmt.Printf("  FEN: %s\n\n", entry.fenBefore)
			if cpLoss >= appendMinLoss {
				findings = append(findings, replayFinding{
					FEN:      entry.fenBefore,
					Tag:      fmt.Sprintf("lichess-%s-ply%d", gameID, entry.ply),
					Expected: sfBefore.BestMove,
					Bad:      entry.uci,
					Notes:    fmt.Sprintf("%s loss=%dcp sf-depth=%d", strings.ToLower(label), cpLoss, sfDepth),
				})
			}
		}
	}
	fmt.Printf("\n=== Summary: %d %s bot half-moves ===\n", totalBotMoves, colorName(botColor))
	fmt.Printf("  Blunders     (>=%d cp): %d\n", BlunderThreshold, totalBlunders)
	fmt.Printf("  Mistakes     (>=%d cp): %d\n", mistakeThreshold, totalMistakes)
	fmt.Printf("  Inaccuracies (>=%d cp): %d\n", inaccuracyThreshold, totalInaccuracies)
	if cfg.AppendFENsPath != "" {
		if err := appendReplayFindings(cfg.AppendFENsPath, findings); err != nil {
			log.Fatalf("Failed to append FENs to %s: %v", cfg.AppendFENsPath, err)
		}
	}
}

func appendReplayFindings(path string, findings []replayFinding) error {
	if len(findings) == 0 {
		fmt.Printf("No replay FENs met append threshold for %s\n", path)
		return nil
	}
	existing, err := existingBenchFENs(path)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	appended := 0
	for _, finding := range findings {
		if existing[finding.FEN] {
			continue
		}
		line := fmt.Sprintf("%s | %s | %s | %s | %s\n",
			finding.FEN,
			sanitizeBenchField(finding.Tag),
			sanitizeBenchField(finding.Expected),
			sanitizeBenchField(finding.Bad),
			sanitizeBenchField(finding.Notes),
		)
		if _, err := w.WriteString(line); err != nil {
			return err
		}
		existing[finding.FEN] = true
		appended++
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Printf("Appended %d replay FENs to %s (%d duplicates skipped)\n", appended, path, len(findings)-appended)
	return nil
}

func existingBenchFENs(path string) (map[string]bool, error) {
	existing := map[string]bool{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return existing, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		fen := strings.TrimSpace(strings.SplitN(line, "|", 2)[0])
		if fen != "" {
			existing[fen] = true
		}
	}
	return existing, scanner.Err()
}

func sanitizeBenchField(s string) string {
	s = strings.ReplaceAll(s, "|", "/")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func replayUCIAnalysisEntries(moveText string) ([]uciAnalysisEntry, error) {
	tokens := strings.Fields(moveText)
	b := &board.Board{}
	b.ResetDefault()
	side := color.White
	fullMove := 1
	var previousMove *board.LastMove
	entries := make([]uciAnalysisEntry, 0, len(tokens))
	for i, token := range tokens {
		m, err := matchUCIMove(b, side, previousMove, token)
		if err != nil {
			return nil, fmt.Errorf("ply %d %s: %w", i+1, token, err)
		}
		fenBefore := BoardToFEN(b, side, previousMove, fullMove)
		after := b.Copy()
		afterMove := board.MakeMove(&m, after)
		nextSide := side ^ 1
		nextFullMove := fullMove
		if side == color.Black {
			nextFullMove++
		}
		entries = append(entries, uciAnalysisEntry{
			ply:         i + 1,
			fullMoveNum: fullMove,
			side:        side,
			move:        m,
			uci:         MoveToUCI(m),
			fenBefore:   fenBefore,
			fenAfter:    BoardToFEN(after, nextSide, afterMove, nextFullMove),
		})
		b = after
		previousMove = afterMove
		side = nextSide
		fullMove = nextFullMove
	}
	return entries, nil
}

func inferBotColorFromLog(logPath, gameID string) (color.Color, bool) {
	f, err := os.Open(logPath)
	if err != nil {
		return color.White, false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	active := gameID == ""
	sawGameFull := false
	var lastColor color.Color
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `\"type\":\"gameFull\"`) || strings.Contains(line, `"type":"gameFull"`) {
			sawGameFull = true
			if m := reGameFullID.FindStringSubmatch(line); m != nil {
				id := firstNonEmpty(m[1], m[2])
				active = gameID == "" || id == gameID
			} else {
				active = gameID == ""
			}
			continue
		}
		if sawGameFull && !active {
			continue
		}
		m := reAIColor.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if m[1] == "White" {
			lastColor = color.White
		} else {
			lastColor = color.Black
		}
		found = true
	}
	return lastColor, found
}
