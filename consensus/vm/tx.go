/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package vm

import "github.com/Qitmeer/qng/core/types"

type Tx struct {
	Type  types.TxType
	From  string
	To    string
	Value uint64
	Data  []byte
}

func (tx *Tx) GetTxType() types.TxType {
	return tx.Type
}

func (tx *Tx) GetFrom() string {
	return tx.From
}
func (tx *Tx) GetTo() string {
	return tx.To
}
func (tx *Tx) GetValue() uint64 {
	return tx.Value
}
func (tx *Tx) GetData() []byte {
	return tx.Data
}
