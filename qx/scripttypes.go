package qx

import (
	"encoding/hex"
	"encoding/json"
	"github.com/Qitmeer/qng/engine/txscript"
)

const (
	INPUT_NAME  = "input"
	OUTPUT_NAME = "output"
)

type ScriptTypeIndex map[string]map[int]txscript.ScriptClass

func (this *ScriptTypeIndex) InputTypeSet(index int, scripttype txscript.ScriptClass) {
	if _, ok := (*this)[INPUT_NAME]; !ok {
		(*this)[INPUT_NAME] = map[int]txscript.ScriptClass{}
	}
	(*this)[INPUT_NAME][index] = scripttype
}

func (this *ScriptTypeIndex) OutputTypeSet(index int, scripttype txscript.ScriptClass) {
	if _, ok := (*this)[OUTPUT_NAME]; !ok {
		(*this)[OUTPUT_NAME] = map[int]txscript.ScriptClass{}
	}
	(*this)[OUTPUT_NAME][index] = scripttype
}

func (this *ScriptTypeIndex) Encode() (string, error) {
	b, err := json.Marshal(*this)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (this *ScriptTypeIndex) String() string {
	b, err := json.Marshal(*this)
	if err != nil {
		return "default:regular"
	}
	return string(b)
}

func (this *ScriptTypeIndex) FindInputScriptType(index int) txscript.ScriptClass {
	if _, ok := (*this)[INPUT_NAME]; !ok {
		return txscript.PubKeyHashTy
	}
	if _, ok := (*this)[INPUT_NAME][index]; !ok {
		return txscript.PubKeyHashTy
	}
	return (*this)[INPUT_NAME][index]
}

func DecodeScriptTypeIndex(str string) (*ScriptTypeIndex, error) {
	b, err := hex.DecodeString(str)
	if err != nil {
		return &ScriptTypeIndex{}, err
	}
	var r ScriptTypeIndex
	err = json.Unmarshal(b, &r)
	if err != nil {
		return &ScriptTypeIndex{}, err
	}
	return &r, nil
}
