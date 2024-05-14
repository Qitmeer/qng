package meer

import (
	"encoding/hex"

	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/crypto"
)

const MainAllocHash = "204bb2f79e29453543b74b620533c854ddd327603fdcadb5c938b4e29e2cbd0b"
const MixAllocHash = "7630112cc9bbc47f2370d67890d40c98a7bf69d80d09f4c1c156a5ad373d46c6"
const TestAllocHash = "7630112cc9bbc47f2370d67890d40c98a7bf69d80d09f4c1c156a5ad373d46c6"
const PrivAllocHash = "7630112cc9bbc47f2370d67890d40c98a7bf69d80d09f4c1c156a5ad373d46c6"

func BuildGenesisHash(network string) string {
	switch network {
	case params.MainNetParams.Name:
		return hex.EncodeToString(crypto.Keccak256([]byte(mainAllocData)))
	case params.MixNetParams.Name:
		return hex.EncodeToString(crypto.Keccak256([]byte(mixAllocData)))
	case params.TestNetParams.Name:
		return hex.EncodeToString(crypto.Keccak256([]byte(testAllocData)))
	case params.PrivNetParams.Name:
		return hex.EncodeToString(crypto.Keccak256([]byte(privAllocData)))
	default:
		return ""

	}
}
