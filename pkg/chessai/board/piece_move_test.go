package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func buildBoardWithInitialMoves(initialMove *[]location.Move) (*Board, *LastMove) {
	bo1 := Board{}
	bo1.ResetDefault()
	var lastMove *LastMove
	if initialMove != nil {
		for _, m := range *initialMove {
			lastMove = MakeMove(&m, &bo1)
		}
	}
	return &bo1, lastMove
}

func testPieceGetMoves(t *testing.T, l location.Location, initialMove *[]location.Move, expectedMoves int) {
	bo1, _ := buildBoardWithInitialMoves(initialMove)
	if l.GetRow() == StartRow[color.Black]["Piece"] {
		assert.Equal(t, color.Black, bo1.GetPiece(l).GetColor())
	} else if l.GetRow() == StartRow[color.White]["Piece"] {
		assert.Equal(t, color.White, bo1.GetPiece(l).GetColor())
	}
	moves := bo1.GetPiece(l).GetMoves(bo1, false)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, expectedMoves, len(*moves))
	}
	moves = bo1.GetPiece(l).GetMoves(bo1, true)
	assert.NotNil(t, moves)
	if moves != nil {
		if expectedMoves > 0 {
			assert.Equal(t, 1, len(*moves))
		}
	}
}

func TestBishopGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(StartRow[color.White]["Piece"], 2), nil, 0)
}

func TestBishopGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(5, 4), &[]location.Move{{
		Start: location.NewLocation(7, 2),
		End:   location.NewLocation(5, 4),
	}}, 7)
}

func TestBishopGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(StartRow[color.Black]["Piece"], 2), nil, 0)
}

func TestBishopGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(2, 4), &[]location.Move{{
		Start: location.NewLocation(0, 2),
		End:   location.NewLocation(2, 4),
	}}, 7)
}

func TestQueenGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(StartRow[color.White]["Piece"], 3), nil, 0)
}

func TestQueenGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(5, 3), &[]location.Move{{
		Start: location.NewLocation(7, 3),
		End:   location.NewLocation(5, 3),
	}}, 18)
}

func TestQueenGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(0, 3), nil, 0)
}

func TestQueenGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(2, 3), &[]location.Move{{
		Start: location.NewLocation(0, 3),
		End:   location.NewLocation(2, 3),
	}}, 18)
}

func TestRookGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(7, 7), nil, 0)
}

func TestRookGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(4, 4), &[]location.Move{{
		Start: location.NewLocation(7, 7),
		End:   location.NewLocation(4, 4),
	}}, 11)
}

func TestRookGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(0, 0), nil, 0)
}

func TestRookGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(3, 3), &[]location.Move{{
		Start: location.NewLocation(0, 0),
		End:   location.NewLocation(3, 3),
	}}, 11)
}

func TestKnightGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(7, 1), nil, 2)
}

func TestKnightGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(3, 6), &[]location.Move{{
		Start: location.NewLocation(7, 1),
		End:   location.NewLocation(3, 6),
	}}, 6)
}

func TestKnightGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(0, 1), nil, 2)
}

func TestKnightGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(4, 6), &[]location.Move{{
		Start: location.NewLocation(0, 1),
		End:   location.NewLocation(4, 6),
	}}, 6)
}

func TestPawnGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(6, 3), nil, 2)
}

func TestPawnGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(4, 3), &[]location.Move{{
		Start: location.NewLocation(6, 3),
		End:   location.NewLocation(4, 3),
	}}, 1)
}

func TestPawnGetMovesAttack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(5, 3), &[]location.Move{{
		Start: location.NewLocation(1, 3),
		End:   location.NewLocation(5, 3),
	}}, 2)
}

func TestPawnGetMovesPromoteAttackWhite(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(1, 3), &[]location.Move{{
		Start: location.NewLocation(6, 3),
		End:   location.NewLocation(1, 3),
	}}, 4)
}

func TestPawnGetMovesPromoteAttackBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(6, 3), &[]location.Move{{
		Start: location.NewLocation(1, 3),
		End:   location.NewLocation(6, 3),
	}}, 4)
}

func TestPawnGetMovesPromoteWhite(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.NewLocation(6, 3),
		End:   location.NewLocation(3, 3),
	}, {
		Start: location.NewLocation(7, 3),
		End:   location.NewLocation(2, 3),
	}, {
		Start: location.NewLocation(7, 2),
		End:   location.NewLocation(4, 2),
	}, {
		Start: location.NewLocation(7, 4),
		End:   location.NewLocation(2, 4),
	}, {
		Start: location.NewLocation(1, 3),
		End:   location.NewLocation(6, 3),
	}})
	moves := bo1.GetPiece(location.NewLocation(6, 3)).GetMoves(bo1, false)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, len(piece.PawnPromotionOptions), len(*moves))
		for i, m := range *moves {
			end := m.GetEnd()
			promotion, promotedType := end.GetPawnPromotion()
			assert.True(t, promotion)
			assert.Equal(t, piece.PawnPromotionOptions[i], promotedType)
		}
	}
	moves = bo1.GetPiece(location.NewLocation(6, 3)).GetMoves(bo1, true)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, 1, len(*moves))
	}
}

func TestGetMovesEnPassantSingleOpportunity(t *testing.T) {
	testEnPassantGetMoves(t, &[]location.Move{
		{
			Start: location.NewLocation(6, 3),
			End:   location.NewLocation(3, 3),
		},
		{
			Start: location.NewLocation(1, 4),
			End:   location.NewLocation(3, 4),
		},
	}, 1)
}

func TestPawnGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(1, 1), nil, 2)
}

func TestPawnGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(3, 1), &[]location.Move{{
		Start: location.NewLocation(1, 1),
		End:   location.NewLocation(3, 1),
	}}, 1)
}

func TestPawnGetMovesBlackAttack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(5, 3), &[]location.Move{{
		Start: location.NewLocation(1, 3),
		End:   location.NewLocation(5, 3),
	}}, 2)
}

func TestKingGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(7, 4), nil, 0)
}

func TestKingGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(4, 4), &[]location.Move{{
		Start: location.NewLocation(7, 4),
		End:   location.NewLocation(4, 4),
	}}, 8)
}

func TestKingGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(0, 4), nil, 0)
}

func TestKingGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(3, 4), &[]location.Move{{
		Start: location.NewLocation(0, 4),
		End:   location.NewLocation(3, 4),
	}}, 8)
}

func TestKingGetMovesDefended(t *testing.T) {
	testPieceGetMoves(t, location.NewLocation(4, 4), &[]location.Move{{
		Start: location.NewLocation(0, 4),
		End:   location.NewLocation(4, 4),
	}}, 5)
}

func TestKingCannotMoveIntoCheck(t *testing.T) {
	b, previousMove := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.NewLocation(0, 4),
		End:   location.NewLocation(4, 4),
	}})
	moves := b.GetAllMoves(color.Black, previousMove)
	numKingMoves := 0
	for _, m := range *moves {
		if m.Start.GetRow() == 4 && m.Start.GetCol() == 4 {
			numKingMoves++
		}
	}
	assert.Equal(t, 5, numKingMoves)
}

func TestKingGetMovesCastleLeft(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.NewLocation(0, 3),
		End:   location.NewLocation(3, 3),
	}, {
		Start: location.NewLocation(0, 2),
		End:   location.NewLocation(3, 2),
	}, {
		Start: location.NewLocation(0, 1),
		End:   location.NewLocation(3, 1),
	}})
	moves := bo1.GetPiece(location.NewLocation(0, 4)).GetMoves(bo1, true)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, 1, len(*moves))
	}
	moves = bo1.GetPiece(location.NewLocation(0, 4)).GetMoves(bo1, false)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, 2, len(*moves))
		MakeMove(&(*moves)[1], bo1)
		assert.False(t, bo1.IsEmpty(location.NewLocation(0, 2)))
		assert.Equal(t, piece.KingType,
			bo1.GetPiece(location.NewLocation(0, 2)).GetPieceType(),
		)
		assert.False(t, bo1.IsEmpty(location.NewLocation(0, 3)))
		assert.Equal(t, piece.RookType,
			bo1.GetPiece(location.NewLocation(0, 3)).GetPieceType(),
		)
	}
}

func TestKingGetMovesCastleRight(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.NewLocation(0, 5),
		End:   location.NewLocation(3, 5),
	}, {
		Start: location.NewLocation(0, 6),
		End:   location.NewLocation(3, 6),
	}})
	moves := bo1.GetPiece(location.NewLocation(0, 4)).GetMoves(bo1, true)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, 1, len(*moves))
	}
	moves = bo1.GetPiece(location.NewLocation(0, 4)).GetMoves(bo1, false)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, 2, len(*moves))
		MakeMove(&(*moves)[1], bo1)
		assert.False(t, bo1.IsEmpty(location.NewLocation(0, 6)))
		assert.Equal(t, piece.KingType,
			bo1.GetPiece(location.NewLocation(0, 6)).GetPieceType(),
		)
		assert.False(t, bo1.IsEmpty(location.NewLocation(0, 5)))
		assert.Equal(t, piece.RookType,
			bo1.GetPiece(location.NewLocation(0, 5)).GetPieceType(),
		)
	}
}
