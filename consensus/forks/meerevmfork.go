package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

const (
	// What main height can transfer the locked utxo in genesis to MeerVM
	MeerEVMForkMainHeight = 959000

	// 21024000000000000 (Total)-5051813000000000 (locked genesis)-2138713274285398 (meerevm genesis) = 13833473725714602
	MeerEVMForkTotalSubsidy = 13833473725714602

	// subsidy reduction interval
	SubsidyReductionInterval = 7358400
)

func IsMeerEVMFork(tx *types.Transaction, ip *types.TxInput, mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	if mainHeight < MeerEVMForkMainHeight {
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

func IsMeerEVMForkHeight(mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	return mainHeight >= MeerEVMForkMainHeight
}
