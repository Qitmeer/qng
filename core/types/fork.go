package types

import "math"

const (
	// crosschain export tx fork for genesis locked utxo
	ExportMaxLockUTXOFork = math.MaxInt64
)

func IsExportUTXOForkInput(ip *TxInput) bool {
	return ip.AmountIn.Value == ExportMaxLockUTXOFork
}
