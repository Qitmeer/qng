package forks

import (
	mparams "github.com/Qitmeer/qng/meerevm/params"
)

const (
	// New gas limit for evm block
	GasLimitForkEVMHeight = 606567
)

func IsGasLimitForkHeight(number int64, chainID int64) bool {
	if chainID != mparams.QngMainnetChainConfig.ChainID.Int64() {
		return false
	}
	return number >= GasLimitForkEVMHeight
}

func NeedFixedGasLimit(number int64, chainID int64) bool {
	if chainID != mparams.QngMainnetChainConfig.ChainID.Int64() {
		return false
	}
	return !IsGasLimitForkHeight(number, chainID)
}
