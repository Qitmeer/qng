package qx

import (
	"encoding/hex"
	"encoding/json"
	"github.com/Qitmeer/qng/core/types"
)

const (
	INPUT_NAME  = "input"
	OUTPUT_NAME = "output"
)

type TxTypeIndex map[string]map[int]types.TxType

func (this *TxTypeIndex) InputTypeSet(index int, txtype types.TxType) {
	if _, ok := (*this)[INPUT_NAME]; !ok {
		(*this)[INPUT_NAME] = map[int]types.TxType{}
	}
	(*this)[INPUT_NAME][index] = txtype
}

func (this *TxTypeIndex) OutputTypeSet(index int, txtype types.TxType) {
	if _, ok := (*this)[OUTPUT_NAME]; !ok {
		(*this)[OUTPUT_NAME] = map[int]types.TxType{}
	}
	(*this)[OUTPUT_NAME][index] = txtype
}

func (this *TxTypeIndex) Encode() (string, error) {
	b, err := json.Marshal(*this)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (this *TxTypeIndex) FindInputTxType(index int) types.TxType {
	if _, ok := (*this)[INPUT_NAME]; !ok {
		return types.TxTypeRegular
	}
	if _, ok := (*this)[INPUT_NAME][index]; !ok {
		return types.TxTypeRegular
	}
	return (*this)[INPUT_NAME][index]
}

func DecodeTxTypeIndex(str string) (*TxTypeIndex, error) {
	b, err := hex.DecodeString(str)
	if err != nil {
		return &TxTypeIndex{}, err
	}
	var r TxTypeIndex
	err = json.Unmarshal(b, &r)
	if err != nil {
		return &TxTypeIndex{}, err
	}
	return &r, nil
}
