package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

var possibleMoves = []location.RelativeLocation{
	{-2, 1},
	{-1, 2},
	{1, 2},
	{2, 1},
	{2, -1},
	{1, -2},
	{-2, -1},
	{-1, -2},
}

type Knight struct {
	Location location.Location
	Color    color.Color
}

func (r *Knight) GetChar() rune {
	return piece.KnightChar
}

func (r *Knight) GetPieceType() byte {
	return piece.KnightType
}

func (r *Knight) GetColor() color.Color {
	return r.Color
}

func (r *Knight) SetColor(color color.Color) {
	r.Color = color
}

func (r *Knight) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Knight) GetPosition() location.Location {
	return r.Location
}

/**
 * Determines the next locations which a knight can move to.
 * TODO (Devan) cache lookups
 */
func (r *Knight) getNextLocations(board *Board) *[]location.Location {
	var locations []location.Location
	for _, possibleMove := range possibleMoves {
		loc := r.GetPosition()
		loc, inBounds := loc.AddRelative(possibleMove)
		if inBounds {
			locations = append(locations, loc)
		}
	}
	return &locations
}

/**
 * Calculates all valid moves that a knight can make.
 */
func (r *Knight) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	locations := r.getNextLocations(board)
	for _, loc := range *locations {
		pieceOnLocation := board.GetPiece(loc)
		if pieceOnLocation == nil || pieceOnLocation.GetColor() != r.Color {
			possibleMove := location.Move{Start: r.GetPosition(), End: loc}
			if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
				moves = append(moves, location.Move{Start: r.GetPosition(), End: loc})
				if onlyFirstMove {
					return &moves
				}
			}
		}
	}
	return &moves
}

/**
 * Retrieves all squares that this knight can attack.
 */
func (r *Knight) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	locations := r.getNextLocations(board)
	for _, loc := range *locations {
		SetLocationAttackable(attackableBoard, loc)
	}
	return attackableBoard
}

func (r *Knight) Move(m *location.Move, b *Board) {}
