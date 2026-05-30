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
)

type logEntry struct {
	plyNum   int
	clr      color.Color
	fromRow  uint8
	fromCol  uint8
	toRow    uint8
	toCol    uint8
	aiScore  int
	hasScore bool
	grid     [board.Height]string // raw "|"-separated piece strings per row
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
					plyNum:  ply,
					clr:     clr,
					fromRow: lastFromRow,
					fromCol: lastFromCol,
					toRow:   lastToRow,
					toCol:   lastToCol,
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
		movedPiece.SetPosition(fromLoc)
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
				rook.SetPosition(rookFrom)
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
				rook.SetPosition(rookFrom)
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

// RunLogReplay parses the internal lichess game log, replays each AI move through
// local Stockfish, and prints a blunder report. The log typically contains only
// one side's moves (the side the AI is playing); White moves from the opponent
// are not logged.
//
// Usage: ./main log-replay [logPath] [sfDepth] [stockfishPath]
func RunLogReplay(logPath, stockfishPath string, sfDepth int) {
	entries, err := parseLichessLog(logPath)
	if err != nil {
		log.Fatalf("Failed to parse log %s: %v", logPath, err)
	}
	if len(entries) == 0 {
		fmt.Println("No moves found in log.")
		return
	}

	fmt.Printf("Parsed %d moves from %s\n\n", len(entries), logPath)

	sf, err := NewStockfishEngine(stockfishPath)
	if err != nil {
		log.Fatalf("Cannot start Stockfish: %v", err)
	}
	defer sf.Close()

	totalBlunders, totalMistakes, totalInaccuracies := 0, 0, 0

	for i, entry := range entries {
		// Use the color recorded in each entry — the log may contain games where
		// the AI plays White in one game and Black in another.
		entryColor := entry.clr

		// Board after AI's move (B_n), active color = opponent
		bn := boardFromGrid(entry.grid)

		// Derive fullmove number from ply
		fullMove := (entry.plyNum + 1) / 2

		// Reconstruct position before AI's move
		var bPrevGrid *[board.Height]string
		if i > 0 && entries[i-1].clr == entryColor {
			// Only use prev grid for capture restoration if it's the same color
			// (same game, same side).  Across game boundaries the grid belongs to
			// a different game and would give wrong capture information.
			bPrevGrid = &entries[i-1].grid
		}
		wn := unApplyMove(bn, entry.fromRow, entry.fromCol, entry.toRow, entry.toCol,
			entryColor, &entry.grid, bPrevGrid)

		// FEN before AI's move: AI's color is to move
		fenBefore := BoardToFEN(wn, entryColor, nil, fullMove)
		sfBefore := sf.Analyze(fenBefore, sfDepth)

		// Build UCI for what the AI played
		fromLoc := location.NewLocation(entry.fromRow, entry.fromCol)
		toLoc := location.NewLocation(entry.toRow, entry.toCol)
		// Add promotion if pawn reaches back rank
		if movingP := wn.GetPiece(fromLoc); movingP != nil && movingP.GetPieceType() == piece.PawnType {
			if entry.toRow == board.StartRow[entryColor^1]["Piece"] {
				toLoc = toLoc.CreatePawnPromotion(piece.QueenType)
			}
		}
		engineMove := location.Move{Start: fromLoc, End: toLoc}
		engineUCI := MoveToUCI(engineMove)

		// FEN after AI's move: opponent's turn
		opponent := entryColor ^ 1
		fenAfter := BoardToFEN(bn, opponent, nil, fullMove)
		sfAfter := sf.Analyze(fenAfter, sfDepth)

		// centipawn loss for the side that moved (AI)
		cpSTMBefore := sfBefore.CentipawnsSTM
		cpSTMAfterOpponent := sfAfter.CentipawnsSTM
		cpLoss := cpSTMBefore + cpSTMAfterOpponent
		if cpLoss < 0 || engineUCI == sfBefore.BestMove {
			// Clamp negatives (noise) and moves that match SF's recommendation
			// (apparent loss is a depth-consistency artifact, not a real error).
			cpLoss = 0
		}
		// Cap at 2000 cp so mate-score arithmetic doesn't produce misleading large values.
		if cpLoss > 2000 {
			cpLoss = 2000
		}

		label := classifyLoss(cpLoss, engineUCI, sfBefore.BestMove)
		switch label {
		case "BLUNDER":
			totalBlunders++
		case "MISTAKE":
			totalMistakes++
		case "INACCURACY":
			totalInaccuracies++
		}

		// Format move number for display
		moveLabel := fmt.Sprintf("%d.", fullMove)
		if entryColor == color.Black {
			moveLabel = fmt.Sprintf("%d...", fullMove)
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
			entry.plyNum, moveLabel, engineUCI, sfEvalStr, cpLoss, sfBefore.BestMove, aiEvalStr, flagStr)

		if label == "BLUNDER" || label == "MISTAKE" {
			fmt.Printf("  AI played %-6s but SF recommends %-6s  (diff: %+d cp)\n", engineUCI, sfBefore.BestMove, cpLoss)
			fmt.Printf("  FEN: %s\n\n", fenBefore)
		}
	}

	total := len(entries)
	fmt.Printf("\n=== Summary: %d AI half-moves ===\n", total)
	fmt.Printf("  Blunders     (>=%d cp): %d\n", BlunderThreshold, totalBlunders)
	fmt.Printf("  Mistakes     (>=%d cp): %d\n", mistakeThreshold, totalMistakes)
	fmt.Printf("  Inaccuracies (>=%d cp): %d\n", inaccuracyThreshold, totalInaccuracies)
}
