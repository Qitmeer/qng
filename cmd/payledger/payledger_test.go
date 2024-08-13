package main

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/testutils"
	"testing"
)

func TestLockedLedger(t *testing.T) {
	node, err := testutils.StartMockNode(nil)
	if err != nil {
		t.Error(err)
	}
	defer node.Stop()

	genesisTxHash := params.ActiveNetParams.Params.GenesisBlock.Transactions()[1].Hash()
	testutils.AssertBlockOrderHeightTotal(t, node, 1, 1, 0)
	spendAmt := types.Amount{Value: 50000 * types.AtomsPerCoin, Id: types.MEERA}

	lockTime := int64(2)
	txid, addr := testutils.SpendUtxo(t, node, types.NewOutPoint(genesisTxHash, 61), spendAmt, lockTime)
	t.Logf("[%v]: tx %v which spend %v has been sent, address:%s", node.ID(), txid.TxHash().String(), spendAmt.String(), addr.String())
}
