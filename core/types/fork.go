package types

import "math"

const (
	// crosschain export tx fork for genesis locked utxo
	MeerEVMForkInput = math.MaxInt64
)

func IsMeerEVMForkInput(ip *TxInput) bool {
	return ip.AmountIn.Value == MeerEVMForkInput
}
