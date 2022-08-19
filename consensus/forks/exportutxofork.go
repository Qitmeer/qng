package forks

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

func IsExportUTXOFork(tx *types.Transaction, ip *types.TxInput, mainHeight int64) bool {
	if params.ActiveNetParams.ExportUTXOForkMainHeight == 0 {
		return false
	}
	if mainHeight < params.ActiveNetParams.ExportUTXOForkMainHeight {
		return false
	}
	if !types.IsCrossChainExportTx(tx) {
		return false
	}
	return IsMaxLockUTXOInGenesis(&ip.PreviousOut)
}

func IsMaxLockUTXOInGenesis(op *types.TxOutPoint) bool {
	gblock := params.ActiveNetParams.GenesisBlock
	for _, tx := range gblock.Transactions {
		if tx.CachedTxHash().IsEqual(&op.Hash) {
			if op.OutIndex >= uint32(len(tx.TxOut)) {
				return false
			}
			ops, err := txscript.ParseScript(tx.TxOut[op.OutIndex].PkScript)
			if err != nil {
				return false
			}
			if ops[1].GetOpcode().GetValue() != txscript.OP_CHECKLOCKTIMEVERIFY {
				return false
			}
			lockTime := txscript.GetInt64FromOpcode(ops[0])
			if lockTime == params.ActiveNetParams.LedgerParams.MaxLockHeight {
				return true
			}
			return false
		}
	}
	return false
}
