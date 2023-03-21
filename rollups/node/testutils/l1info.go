package testutils

import (
	"math/big"
	"math/rand"

	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qit/common"
	"github.com/Qitmeer/qit/core/types"
)

type MockBlockInfo struct {
	// Prefixed all fields with "Info" to avoid collisions with the interface method names.

	InfoHash        common.Hash
	InfoParentHash  common.Hash
	InfoCoinbase    common.Address
	InfoRoot        common.Hash
	InfoNum         uint64
	InfoTime        uint64
	InfoMixDigest   [32]byte
	InfoBaseFee     *big.Int
	InfoReceiptRoot common.Hash
	InfoGasUsed     uint64
}

func (l *MockBlockInfo) Hash() common.Hash {
	return l.InfoHash
}

func (l *MockBlockInfo) ParentHash() common.Hash {
	return l.InfoParentHash
}

func (l *MockBlockInfo) Coinbase() common.Address {
	return l.InfoCoinbase
}

func (l *MockBlockInfo) Root() common.Hash {
	return l.InfoRoot
}

func (l *MockBlockInfo) NumberU64() uint64 {
	return l.InfoNum
}

func (l *MockBlockInfo) Time() uint64 {
	return l.InfoTime
}

func (l *MockBlockInfo) MixDigest() common.Hash {
	return l.InfoMixDigest
}

func (l *MockBlockInfo) BaseFee() *big.Int {
	return l.InfoBaseFee
}

func (l *MockBlockInfo) ReceiptHash() common.Hash {
	return l.InfoReceiptRoot
}

func (l *MockBlockInfo) GasUsed() uint64 {
	return l.InfoGasUsed
}

func (l *MockBlockInfo) ID() eth.BlockID {
	return eth.BlockID{Hash: l.InfoHash, Number: l.InfoNum}
}

func (l *MockBlockInfo) BlockRef() eth.L1BlockRef {
	return eth.L1BlockRef{
		Hash:       l.InfoHash,
		Number:     l.InfoNum,
		ParentHash: l.InfoParentHash,
		Time:       l.InfoTime,
	}
}

func RandomBlockInfo(rng *rand.Rand) *MockBlockInfo {
	return &MockBlockInfo{
		InfoParentHash:  RandomHash(rng),
		InfoNum:         rng.Uint64(),
		InfoTime:        rng.Uint64(),
		InfoHash:        RandomHash(rng),
		InfoBaseFee:     big.NewInt(rng.Int63n(1000_000 * 1e9)), // a million GWEI
		InfoReceiptRoot: types.EmptyRootHash,
		InfoRoot:        RandomHash(rng),
		InfoGasUsed:     rng.Uint64(),
	}
}

func MakeBlockInfo(fn func(l *MockBlockInfo)) func(rng *rand.Rand) *MockBlockInfo {
	return func(rng *rand.Rand) *MockBlockInfo {
		l := RandomBlockInfo(rng)
		if fn != nil {
			fn(l)
		}
		return l
	}
}
