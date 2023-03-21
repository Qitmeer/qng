package test

import (
	"math/rand"

	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qng/rollups/node/rollup/derive"
	"github.com/Qitmeer/qng/rollups/node/testutils"
	"github.com/Qitmeer/qit/core/types"
	"github.com/Qitmeer/qit/trie"
)

// RandomL2Block returns a random block whose first transaction is a random
// L1 Info Deposit transaction.
func RandomL2Block(rng *rand.Rand, txCount int) (*types.Block, []*types.Receipt) {
	l1Block := types.NewBlock(testutils.RandomHeader(rng),
		nil, nil, nil, trie.NewStackTrie(nil))
	l1InfoTx, err := derive.L1InfoDeposit(0, l1Block, eth.SystemConfig{}, testutils.RandomBool(rng))
	if err != nil {
		panic("L1InfoDeposit: " + err.Error())
	}
	return testutils.RandomBlockPrependTxs(rng, txCount, types.NewTx(l1InfoTx))
}
