package forks

import (
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
)

const (
	// MeerEVM is enabled  and new subsidy calculation
	MeerEVMForkMainHeight = 951100

	// What main height can transfer the locked utxo in genesis to MeerEVM
	// Must after MeerEVMForkMainHeight
	MeerEVMUTXOUnlockMainHeight = 1200000

	// 21024000000000000 (Total)-5051813000000000 (locked genesis)-1215912000000000 (meerevm genesis) = 14756275000000000
	MeerEVMForkTotalSubsidy = 14756275000000000

	// subsidy reduction interval  48 days
	SubsidyReductionInterval = 138240

	// Subsidy reduction multiplier.
	MulSubsidy = 100

	// Subsidy reduction divisor.
	DivSubsidy = 101

	// TODO Temporary code to be deleted later
	BadBlockHashHex = "f258088b0422b2a52f8c0aae7225fc15b8508e33cf17f873e88a51fae634d45d"
)

func IsVaildEVMUTXOUnlockTx(tx *types.Transaction, ip *types.TxInput, mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	if mainHeight < MeerEVMForkMainHeight ||
		mainHeight < MeerEVMUTXOUnlockMainHeight {
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

func IsMeerEVMUTXOHeight(mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	return mainHeight >= MeerEVMUTXOUnlockMainHeight
}

func IsBeforeMeerEVMForkHeight(mainHeight int64) bool {
	if params.ActiveNetParams.Net != protocol.MainNet {
		return false
	}
	return mainHeight < MeerEVMForkMainHeight
}
