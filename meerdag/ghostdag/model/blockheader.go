package model

import (
	"github.com/Qitmeer/qng/core/types/pow"
)

type BlockHeader interface {
	Bits() uint32
	Pow() pow.IPow
}
