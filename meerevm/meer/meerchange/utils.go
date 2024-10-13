package meerchange

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eparams "github.com/ethereum/go-ethereum/params"
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
	switch params.ActiveNetParams.Net {
	case protocol.MainNet:
		return eparams.QngMainnetChainConfig.ChainID
	case protocol.TestNet:
		return eparams.QngTestnetChainConfig.ChainID
	case protocol.MixNet:
		return eparams.QngMixnetChainConfig.ChainID
	case protocol.PrivNet:
		return eparams.QngPrivnetChainConfig.ChainID
	}
	return nil
}

func EnableContractAddr() {
	ContractAddr = common.HexToAddress(CONTRACTADDRESS)
}

func DisableContractAddr() {
	ContractAddr = common.Address{}
	Bytecode = nil
}
