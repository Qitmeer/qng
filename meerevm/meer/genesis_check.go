package meer

import (
	"encoding/hex"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/crypto"
)

const MainAllocHash = "e36181b59aaec0602dbec5e16570e60ca3828abe07bf79fd08d3ca379cdae425"
const MixAllocHash = "c877cb5688d5daf9e7b20eba0c46fec0bf096fb06d966aa601068c7e7b795e86"
const TestAllocHash = "c877cb5688d5daf9e7b20eba0c46fec0bf096fb06d966aa601068c7e7b795e86"
const PrivAllocHash = "c877cb5688d5daf9e7b20eba0c46fec0bf096fb06d966aa601068c7e7b795e86"

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
