package acct

import (
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
)

type UTXOOP struct {
	add   bool
	op    *types.TxOutPoint
	entry *blockchain.UtxoEntry
}
