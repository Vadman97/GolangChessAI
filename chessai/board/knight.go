package board

import (
	"ChessAI3/chessai/board/piece"
)

var possibleMoves = []Location{
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
	Location Location
	Color    byte
}

func (r *Knight) GetChar() rune {
	return piece.KnightChar
}

func (r *Knight) GetPieceType() byte {
	return piece.KnightType
}

func (r *Knight) GetColor() byte {
	return r.Color
}

func (r *Knight) SetColor(color byte) {
	r.Color = color
}

func (r *Knight) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Knight) GetPosition() Location {
	return r.Location
}

/**
 * Determines the next locations which a knight can move to.
 * TODO (Devan) cache lookups
 */
func (r *Knight) getNextLocations(board *Board) *[]Location {
	var locations []Location
	for _, possibleMove := range possibleMoves {
		location := r.GetPosition()
		location = location.Add(possibleMove)
		if location.InBounds() {
			locations = append(locations, location)
		}
	}
	return &locations
}

/**
 * Calculates all valid moves that a knight can make.
 */
func (r *Knight) GetMoves(board *Board) *[]Move {
	var moves []Move
	locations := r.getNextLocations(board)
	for _, location := range *locations {
		pieceOnLocation := board.GetPiece(location)
		if pieceOnLocation == nil || pieceOnLocation.GetColor() != r.Color {
			moves = append(moves, Move{r.GetPosition(), location})
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
	for _, location := range *locations {
		SetLocationAttackable(attackableBoard, location)
	}
	return attackableBoard
}

func (r *Knight) Move(m *Move, b *Board) {}
