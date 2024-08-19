package meerchange

import (
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func IsMeerChangeTx(tx *types.Transaction) bool {
	if len(params.ActiveNetParams.MeerChangeContractAddr) <= 0 {
		return false
	}
	if tx == nil {
		return false
	}
	if tx.To() == nil {
		return false
	}
	return *tx.To() == common.HexToAddress(params.ActiveNetParams.MeerChangeContractAddr)
}
