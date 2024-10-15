package forks

import (
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

func GetCancunForkDifficulty(number *big.Int) *big.Int {
	if params.ActiveNetParams.IsCancunFork(number) {
		return common.Big0
	}
	return common.Big1
}
