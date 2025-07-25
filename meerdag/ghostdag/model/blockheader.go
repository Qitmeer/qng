package model

import (
	"github.com/Qitmeer/qng/v2/consensus/engine/pow"
)

type BlockHeader interface {
	Bits() uint32
	Pow() pow.IPow
}
