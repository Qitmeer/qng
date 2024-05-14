package meer

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/crypto"
)

const MainAllocHash = "e36181b59aaec0602dbec5e16570e60ca3828abe07bf79fd08d3ca379cdae425"
const MixAllocHash = "5cc4ee79533fa24494c44d391dabb7816b39a3c4ac9f0e34b2a1877764aaac9f"
const TestAllocHash = "5cc4ee79533fa24494c44d391dabb7816b39a3c4ac9f0e34b2a1877764aaac9f"
const PrivAllocHash = "5cc4ee79533fa24494c44d391dabb7816b39a3c4ac9f0e34b2a1877764aaac9f"

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
