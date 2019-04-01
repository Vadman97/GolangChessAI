package test

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"testing"
)

func buildBoardWithInitialMoves(initialMove *[]location.Move) (*board.Board, *board.LastMove) {
	bo1 := board.Board{}
	bo1.ResetDefault()
	var lastMove *board.LastMove
	if initialMove != nil {
		for _, m := range *initialMove {
			lastMove = board.MakeMove(&m, &bo1)
		}
	}
	return &bo1, lastMove
}

func testPieceGetMoves(t *testing.T, l location.Location, initialMove *[]location.Move, expectedMoves int) {
	bo1, _ := buildBoardWithInitialMoves(initialMove)
	if l.Row == 0 {
		assert.Equal(t, color.Black, bo1.GetPiece(l).GetColor())
	} else if l.Row == 7 {
		assert.Equal(t, color.White, bo1.GetPiece(l).GetColor())
	}
	moves := bo1.GetPiece(l).GetMoves(bo1)
	assert.NotNil(t, moves)
	if moves != nil {
		assert.Equal(t, expectedMoves, len(*moves))
	}
}

func testEnPassantGetMoves(t *testing.T, initialMove *[]location.Move, expectedMoves int) {
	bo1, lastMove := buildBoardWithInitialMoves(initialMove)
	c := (*lastMove.Piece).GetColor()
	c ^= 1
	moves := bo1.GetEnPassantMoves(c, lastMove)
	assert.Equal(t, expectedMoves, len(*moves))
}

func TestBishopGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 7, Col: 2}, nil, 0)
}

func TestBishopGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 5, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 7, Col: 2},
		End:   location.Location{Row: 5, Col: 4},
	}}, 7)
}

func TestBishopGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 0, Col: 2}, nil, 0)
}

func TestBishopGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 2, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 2},
		End:   location.Location{Row: 2, Col: 4},
	}}, 7)
}

func TestQueenGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 7, Col: 3}, nil, 0)
}

func TestQueenGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 5, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 7, Col: 3},
		End:   location.Location{Row: 5, Col: 3},
	}}, 18)
}

func TestQueenGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 0, Col: 3}, nil, 0)
}

func TestQueenGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 2, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 3},
		End:   location.Location{Row: 2, Col: 3},
	}}, 18)
}

func TestRookGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 7, Col: 7}, nil, 0)
}

func TestRookGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 7, Col: 7},
		End:   location.Location{Row: 4, Col: 4},
	}}, 11)
}

func TestRookGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 0, Col: 0}, nil, 0)
}

func TestRookGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 3, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 0},
		End:   location.Location{Row: 3, Col: 3},
	}}, 11)
}

func TestKnightGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 7, Col: 1}, nil, 2)
}

func TestKnightGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 3, Col: 6}, &[]location.Move{{
		Start: location.Location{Row: 7, Col: 1},
		End:   location.Location{Row: 3, Col: 6},
	}}, 6)
}

func TestKnightGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 0, Col: 1}, nil, 2)
}

func TestKnightGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 6}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 1},
		End:   location.Location{Row: 4, Col: 6},
	}}, 6)
}

func TestPawnGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 6, Col: 3}, nil, 2)
}

func TestPawnGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 6, Col: 3},
		End:   location.Location{Row: 4, Col: 3},
	}}, 1)
}

func TestPawnGetMovesAttack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 2, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 6, Col: 3},
		End:   location.Location{Row: 2, Col: 3},
	}}, 2)
}

func TestGetMovesEnPassantSingleOpportunity(t *testing.T) {
	testEnPassantGetMoves(t, &[]location.Move{
		{
			Start: location.Location{Row: 6, Col: 3},
			End:   location.Location{Row: 3, Col: 3},
		},
		{
			Start: location.Location{Row: 1, Col: 4},
			End:   location.Location{Row: 3, Col: 4},
		},
	}, 1)
}

func TestGetMovesEnPassantDoubleOpportunity(t *testing.T) {
	testEnPassantGetMoves(t, &[]location.Move{
		{
			Start: location.Location{Row: 6, Col: 3},
			End:   location.Location{Row: 3, Col: 3},
		},
		{
			Start: location.Location{Row: 6, Col: 5},
			End:   location.Location{Row: 3, Col: 5},
		},
		{
			Start: location.Location{Row: 1, Col: 4},
			End:   location.Location{Row: 3, Col: 4},
		},
	}, 2)
}

func TestGetMovesEnPassantSameColor(t *testing.T) {
	testEnPassantGetMoves(t, &[]location.Move{
		{
			Start: location.Location{Row: 1, Col: 4},
			End:   location.Location{Row: 3, Col: 4},
		},
		{
			Start: location.Location{Row: 1, Col: 2},
			End:   location.Location{Row: 3, Col: 2},
		},
		{
			Start: location.Location{Row: 1, Col: 3},
			End:   location.Location{Row: 3, Col: 3},
		},
	}, 0)
}

func TestGetMovesEnPassantMissedOpportunity(t *testing.T) {
	testEnPassantGetMoves(t, &[]location.Move{
		{
			Start: location.Location{Row: 6, Col: 3},
			End:   location.Location{Row: 3, Col: 3},
		},
		{
			Start: location.Location{Row: 1, Col: 4},
			End:   location.Location{Row: 3, Col: 4},
		},
		{
			Start: location.Location{Row: 6, Col: 5},
			End:   location.Location{Row: 3, Col: 5},
		},
	}, 0)
}

func TestGetMovesBlackEnPassant(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 2}, &[]location.Move{
		{
			Start: location.Location{Row: 1, Col: 2},
			End:   location.Location{Row: 4, Col: 2},
		}, {
			Start: location.Location{Row: 6, Col: 1},
			End:   location.Location{Row: 4, Col: 1},
		},
	}, 1)
}

func TestPawnGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 1, Col: 1}, nil, 2)
}

func TestPawnGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 3, Col: 1}, &[]location.Move{{
		Start: location.Location{Row: 1, Col: 1},
		End:   location.Location{Row: 3, Col: 1},
	}}, 1)
}

func TestPawnGetMovesBlackAttack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 5, Col: 3}, &[]location.Move{{
		Start: location.Location{Row: 1, Col: 3},
		End:   location.Location{Row: 5, Col: 3},
	}}, 2)
}

func TestKingGetMovesStart(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 7, Col: 4}, nil, 0)
}

func TestKingGetMoves(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 7, Col: 4},
		End:   location.Location{Row: 4, Col: 4},
	}}, 8)
}

func TestKingGetMovesStartBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 0, Col: 4}, nil, 0)
}

func TestKingGetMovesBlack(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 3, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 4},
		End:   location.Location{Row: 3, Col: 4},
	}}, 8)
}

func TestKingGetMovesDefended(t *testing.T) {
	testPieceGetMoves(t, location.Location{Row: 4, Col: 4}, &[]location.Move{{
		Start: location.Location{Row: 0, Col: 4},
		End:   location.Location{Row: 4, Col: 4},
	}}, 8)
}

func TestKingCannotMoveIntoCheck(t *testing.T) {
	b, previousMove := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.Location{Row: 0, Col: 4},
		End:   location.Location{Row: 4, Col: 4},
	}})
	moves := b.GetAllMoves(color.Black, previousMove)
	numKingMoves := 0
	for _, m := range *moves {
		if m.Start.Row == 4 && m.Start.Col == 4 {
			numKingMoves++
		}
	}
	assert.Equal(t, 5, numKingMoves)
}

func TestKingGetMovesCastle(t *testing.T) {
	// TODO(Vadim)
}
