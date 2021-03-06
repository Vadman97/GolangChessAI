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

var PawnPromotionOptions = [...]byte{KnightType, QueenType}

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
