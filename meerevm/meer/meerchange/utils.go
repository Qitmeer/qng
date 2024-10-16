package meerchange

import (
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

const (
	funcSigHashLen  = 4
	Version         = 0
	CONTRACTADDRESS = "0x7D698C4E800dBc1E9B7e915BefeDdB59Aa9E8BB6"
)

var (
	ContractAddr common.Address // runtime address expect equal to CONTRACTADDRESS
	Bytecode     []byte
)

func IsDirectMeerChangeTx(tx *types.Transaction) bool {
	if ContractAddr == (common.Address{}) {
		return false
	}
	if tx == nil {
		return false
	}
	if tx.To() == nil {
		return false
	}
	return *tx.To() == ContractAddr
}

func IsExportTx(tx *types.Transaction) bool {
	if IsMeerChangeExportTx(tx) {
		return true
	}
	return IsEntrypointExportTx(tx)
}

func IsMeerChangeTx(tx *types.Transaction) bool {
	if IsDirectMeerChangeTx(tx) {
		return true
	}
	return IsEntrypointMeerChangeTx(tx)
}

func GetChainID() *big.Int {
	return params.ActiveNetParams.MeerConfig.ChainID
}

func EnableContractAddr() {
	ContractAddr = common.HexToAddress(CONTRACTADDRESS)
}

func DisableContractAddr() {
	ContractAddr = common.Address{}
	Bytecode = nil
}
