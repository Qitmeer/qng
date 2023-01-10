package meer

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/crypto"
)

const MainAllocHash = "204bb2f79e29453543b74b620533c854ddd327603fdcadb5c938b4e29e2cbd0b"
const MixAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"
const TestAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"
const PrivAllocHash = "e380c81b956194ce4e38c218f9b99300ef725f58b7ec13963fea763c8379446f"

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
