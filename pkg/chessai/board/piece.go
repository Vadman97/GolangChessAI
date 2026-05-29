package board

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
)

type Piece interface {
	GetChar() rune
	GetColor() byte
	SetColor(byte)
	GetPosition() location.Location
	SetPosition(location.Location)
	GetMoves(board *Board, onlyFirstMove bool) *[]location.Move
	GetAttackableMoves(*Board) BitBoard
	GetPieceType() byte
	Move(m *location.Move, b *Board)
}

func MakeMove(m *location.Move, b *Board) *LastMove {
	// no UnMove function because we delete the piece we destroy
	// easier to store copy of board before making move
	end := m.GetEnd()
	start := m.GetStart()
	if end.Equals(start) {
		panic(fmt.Sprintf("Invalid move attempted! Start and End same: %+v", start))
	} else {
		b.PreviousPositions = append(b.PreviousPositions, b.Hash())
		// piece holds information about its location for convenience
		// game tree stores as compressed game board -> have way to hash compressed game board fast
		// location stored in board coordinates but can be expanded to piece objects
		pieceCaptured := b.GetPiece(end)
		b.move(m)
		pieceMoved := b.GetPiece(end)
		if pieceMoved == nil {
			// Start position had no piece — board is out of sync with the game state.
			panic(fmt.Sprintf("MakeMove: no piece at start %+v (end %+v after move)", start, end))
		}
		pieceMoved.Move(m, b)
		// If a rook was captured from its starting column, invalidate that side's castling right.
		if capturedRook, ok := pieceCaptured.(*Rook); ok {
			capturedRook.Move(&location.Move{Start: end, End: end}, b)
		}

		isCapture := pieceCaptured != nil
		// En passant: pawn moves diagonally to an empty square — the captured pawn
		// is not at the destination but one step behind it (same row as start, same col as end).
		if _, isPawn := pieceMoved.(*Pawn); isPawn && !isCapture && start.GetCol() != end.GetCol() {
			capturedLoc := location.NewLocation(start.GetRow(), end.GetCol())
			if ep := b.GetPiece(capturedLoc); ep != nil {
				if epPawn, ok := ep.(*Pawn); ok && epPawn.GetColor() != pieceMoved.GetColor() {
					b.SetPiece(capturedLoc, nil)
					isCapture = true
				}
			}
		}

		lm := &LastMove{
			Piece:     &pieceMoved,
			Move:      m,
			IsCapture: isCapture,
		}

		// Check for Pawn Promotion
		if hasPromotion, promoteType := end.GetPawnPromotion(); hasPromotion {
			piece := PieceFromType(promoteType)
			piece.SetColor(pieceMoved.GetColor())
			lm.PromotionPiece = &piece
		}

		// here, not in game so that AI can keep track of FiftyMoveDraw condition
		b.UpdateDrawCounter(lm)

		h := b.Hash()
		// check the current position
		for i := len(b.PreviousPositions) - 1; i >= 0; i-- {
			// iterate in reverse because it is faster: more likely to have seen previous position recently
			hash := b.PreviousPositions[i]
			if h == hash {
				// increment keeps track of previous seen count to reduce work - no need to do from scratch
				b.PreviousPositionsSeen++
				break
			}
		}

		return lm
	}
}

func CheckLocationForPiece(pieceColor byte, l location.Location, b *Board) (validMove bool, checkNext bool) {
	data := b.getPieceData(l)
	if data == 0 {
		return true, true // empty: can move here, ray continues
	}
	if data&0x1 != pieceColor {
		return true, false // enemy piece: can capture, ray stops
	}
	return false, false // friendly piece: blocked, ray stops
}

/**
 * Determines if the position is attackable. Attackable means that this position can be seized,
 * regardless of the color of piece on it. This is necessary since King may take a piece, but
 * put itself into check.  This is less strict than CheckLocationForPiece.
 */
func CheckLocationForAttackability(l location.Location, b *Board) bool {
	return b.getPieceData(l) == 0
}

func GetColorTypeRepr(p Piece) string {
	var result string
	if p == nil {
		return "   "
	}
	if p.GetColor() == color.White {
		result += string(color.WhiteChar) + "_"
	} else if p.GetColor() == color.Black {
		result += string(color.BlackChar) + "_"
	}
	return result + string(p.GetChar())
}
