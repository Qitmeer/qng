package txbasetypes

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/params"
	"log"
)

type TxEncodeBase interface {
	AssembleVin(mtx *types.Transaction) error
	AssembleVout(mtx *types.Transaction) error
	SignTx(mtx *types.Transaction) error
	SetTxID(txid string)
	SetOutIndex(outIndex uint32)
	SetSequence(sequence uint32)
	SetAddr(addr string)
	SetAmount(amount types.Amount)
	SetLockTime(lockTime int64)
}

func (this *BaseUTXO) SetTxID(txid string) {
	this.TxID = txid
}

func (this *BaseUTXO) SignTx(mtx *types.Transaction) error {
	return nil
}

func (this *BaseUTXO) SetOutIndex(outIndex uint32) {
	this.OutIndex = outIndex
}

func (this *BaseUTXO) SetSequence(sequence uint32) {
	this.Sequence = sequence
}

func (this *BaseUTXO) SetAddr(addr string) {
	this.Addr = addr
}

func (this *BaseUTXO) SetAmount(amount types.Amount) {
	this.Amount = amount
}

func (this *BaseUTXO) SetLockTime(lockTime int64) {
	this.LockTime = lockTime
}

type BaseUTXO struct {
	TxID     string
	OutIndex uint32
	Sequence uint32
	Amount   types.Amount
	Addr     string
	TxType   types.TxType
	LockTime int64
}

// txtype
func NewTxAssembleVinObject(txid string, outIndex uint32, sequence uint32, locktime int64, txtype types.TxType) TxEncodeBase {
	var utxo TxEncodeBase
	switch txtype {
	case types.TxTypeRegular:
		utxo = &TxTypeRegularUTXO{}
	case types.TxTypeGenesisLock:
		utxo = &TxTypeGenesisLockUTXO{}
	case types.TxTypeCrossChainImport:
		utxo = &TxTypeCrossChainImportUTXO{}
	case types.TxTypeCrossChainExport:
		utxo = &TxTypeCrossChainExportUTXO{}
	default:
		log.Fatalln("txtype not support:", txtype.String())
		return nil
	}
	utxo.SetSequence(sequence)
	utxo.SetTxID(txid)
	utxo.SetOutIndex(outIndex)
	utxo.SetLockTime(locktime)
	return utxo
}

func NewTxAssembleVoutObject(addr string, amount types.Amount, locktime int64, txtype types.TxType) TxEncodeBase {
	var utxo TxEncodeBase
	switch txtype {
	case types.TxTypeRegular:
		utxo = &TxTypeRegularUTXO{}
	case types.TxTypeGenesisLock:
		utxo = &TxTypeGenesisLockUTXO{}
	case types.TxTypeCrossChainImport:
		utxo = &TxTypeCrossChainImportUTXO{}
	case types.TxTypeCrossChainExport:
		utxo = &TxTypeCrossChainExportUTXO{}
	default:
		log.Fatalln("txtype not support:", txtype.String())
		return nil
	}
	utxo.SetAmount(amount)
	utxo.SetAddr(addr)
	utxo.SetLockTime(locktime)
	return utxo
}

type TxSignBase interface {
	Sign(privateKey string, mtx *types.Transaction, inputIndex int, param *params.Params) error
}

func NewTxSignObject(txtype types.TxType) TxSignBase {
	var s TxSignBase
	switch txtype {
	case types.TxTypeRegular:
		s = &TxTypeSignRegular{}
	case types.TxTypeGenesisLock:
		s = &TxTypeSignGenesisBlock{}
	case types.TxTypeCrossChainImport:
		s = &TxTypeSignImport{}
	case types.TxTypeCrossChainExport:
		s = &TxTypeSignExport{}
	default:
		log.Fatalln("unsupport txSign type", txtype.String())
		return nil
	}
	return s
}
