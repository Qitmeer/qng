package meerchange

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	eparams "github.com/ethereum/go-ethereum/params"
	"math/big"
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
