/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import "github.com/Qitmeer/qng/v2/core/types"

type Tx interface {
	GetTxType() types.TxType
	GetFrom() []byte
	GetTo() []byte
	GetValue() uint64
	GetData() []byte
}

type Transaction struct {
	Type  types.TxType
	From  []byte
	To    []byte
	Value uint64
	Data  []byte
}

func (tx *Transaction) GetTxType() types.TxType {
	return tx.Type
}

func (tx *Transaction) GetFrom() []byte {
	return tx.From
}
func (tx *Transaction) GetTo() []byte {
	return tx.To
}
func (tx *Transaction) GetValue() uint64 {
	return tx.Value
}
func (tx *Transaction) GetData() []byte {
	return tx.Data
}
