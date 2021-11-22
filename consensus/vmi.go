package consensus

import "github.com/Qitmeer/qitmeer/consensus"

type VMI interface {
	VerifyTx(tx consensus.Tx) (int64, error)
}
