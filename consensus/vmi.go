package consensus

import "github.com/Qitmeer/qng-core/consensus"

type VMI interface {
	VerifyTx(tx consensus.Tx) (int64, error)
	GetVM(id string) (consensus.ChainVM, error)
}
