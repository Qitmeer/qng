package acct

import (
	"github.com/Qitmeer/qng/v2/core/blockchain/utxo"
	"github.com/Qitmeer/qng/v2/core/types"
)

type UTXOOP struct {
	add   bool
	op    *types.TxOutPoint
	entry *utxo.UtxoEntry
}
