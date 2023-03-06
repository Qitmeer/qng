package utils

import (
	"github.com/Qitmeer/qng/params"
)

// GetNetParams by network name
func GetNetParams(name string) *params.Params {
	switch name {
	case "mainnet":
		return &params.MainNetParams
	case "testnet":
		return &params.TestNetParams
	case "privnet":
		return &params.PrivNetParams
	case "mixnet":
		return &params.MixNetParams
	default:
		return &params.TestNetParams
	}
}
