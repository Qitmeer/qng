package ghostdag

import (
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/meerdag/ghostdag/model"
)

// blockHeader
type blockHeader struct {
	// Difficulty
	bits uint32
	// pow blake2bd | cuckaroo | cuckatoo
	pow pow.IPow
}

func (bh *blockHeader) Bits() uint32 {
	return bh.bits
}

func (bh *blockHeader) Pow() pow.IPow {
	return bh.pow
}

func NewBlockHeader(bits uint32, pow pow.IPow) model.BlockHeader {
	return &blockHeader{
		bits: bits,
		pow:  pow,
	}
}
