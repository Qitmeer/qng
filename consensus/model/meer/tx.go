/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meer

import "github.com/Qitmeer/qng/core/types"

type Tx struct {
	Type  types.TxType
	From  []byte
	To    []byte
	Value uint64
	Data  []byte
}

func (tx *Tx) GetTxType() types.TxType {
	return tx.Type
}

func (tx *Tx) GetFrom() []byte {
	return tx.From
}
func (tx *Tx) GetTo() []byte {
	return tx.To
}
func (tx *Tx) GetValue() uint64 {
	return tx.Value
}
func (tx *Tx) GetData() []byte {
	return tx.Data
}
