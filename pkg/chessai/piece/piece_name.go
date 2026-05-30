package piece

const (
	RookChar   = 'R'
	KnightChar = 'N'
	BishopChar = 'B'
	QueenChar  = 'Q'
	KingChar   = 'K'
	PawnChar   = 'P'
)

const (
	NumPieces  = 7
	NilType    = byte(0)
	RookType   = byte(1)
	KnightType = byte(2)
	BishopType = byte(3)
	QueenType  = byte(4)
	KingType   = byte(5)
	PawnType   = byte(6)
)

// Queen first so the engine prefers it; Rook and Knight cover practical underpromotions.
// BishopType is not included: 2-bit field holds only 3 values, and bishop underpromotion
// is essentially never played (knight covers stalemate-avoidance cases).
var PawnPromotionOptions = [...]byte{QueenType, RookType, KnightType}

var NameToType = map[rune]byte{
	RookChar:   RookType,
	KnightChar: KnightType,
	BishopChar: BishopType,
	QueenChar:  QueenType,
	KingChar:   KingType,
	PawnChar:   PawnType,
}

var TypeToName = map[byte]string{
	RookType:   string(RookChar),
	KnightType: string(KnightChar),
	BishopType: string(BishopChar),
	QueenType:  string(QueenChar),
	KingType:   string(KingChar),
	PawnType:   string(PawnChar),
}
