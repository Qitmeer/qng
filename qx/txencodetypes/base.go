package txencodetypes

import (
	"github.com/Qitmeer/qng/core/types"
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
		utxo = &TxTypeGenesisLockUTXO{}
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
		utxo = &TxTypeGenesisLockUTXO{}
	default:
		log.Fatalln("txtype not support:", txtype.String())
		return nil
	}
	utxo.SetAmount(amount)
	utxo.SetAddr(addr)
	utxo.SetLockTime(locktime)
	return utxo
}
