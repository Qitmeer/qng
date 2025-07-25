package meer

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	qtypes "github.com/Qitmeer/qng/v2/core/types"
	"github.com/ethereum/go-ethereum/core/types"
)

type TxPool interface {
	GetTxs() ([]*qtypes.Tx, []*hash.Hash, error)
	HasTx(h *hash.Hash) bool
	GetSize() int64
	AddTx(tx *qtypes.Tx) (int64, error)
	RemoveTx(tx *qtypes.Tx) error
	CheckMeerChangeExportTx(tx *types.Transaction, ced interface{}, utxoView interface{}) error
	ResetTemplate() error
}
