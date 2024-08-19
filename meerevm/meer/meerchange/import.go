package meerchange

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"strings"
)

func IsMeerChangeImportTx(tx *types.Transaction) bool {
	if !IsMeerChangeTx(tx) {
		return false
	}
	if len(tx.Data()) <= 4 {
		return false
	}
	contractAbi, err := abi.JSON(strings.NewReader(MeerchangeMetaData.ABI))
	if err != nil {
		return false
	}
	method, err := contractAbi.MethodById(tx.Data()[:4])
	if err != nil {
		return false
	}
	if method.Name != (&MeerchangeExport4337Data{}).GetFuncName() {
		return false
	}
	return true
}
