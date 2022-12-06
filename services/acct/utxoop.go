package acct

import (
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/types"
)

type UTXOOP struct {
	add   bool
	op    *types.TxOutPoint
	entry *utxo.UtxoEntry
}
