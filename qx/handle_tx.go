package qx

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/qx/txbasetypes"
	"strings"
	"time"
)

type Input struct {
	TxID       string
	OutIndex   uint32
	SignScript []byte
	sequence   uint32
	InputType  types.TxType
}

type Output struct {
	TargetAddress  string
	Amount         types.Amount
	TargetLockTime int64
	OutputType     types.TxType
}

const MTX_STR_SEPERATE = "-"

func TxEncode(version uint32, lockTime uint32, timestamp *time.Time, inputs []Input, outputs []Output) (string, error) {
	mtx := types.NewTransaction()
	mtx.Version = uint32(version)
	if lockTime != 0 {
		mtx.LockTime = uint32(lockTime)
	}
	if timestamp != nil {
		mtx.Timestamp = *timestamp
	}

	txtypes := &TxTypeIndex{}
	for i, vout := range inputs {
		vinEncode := txbasetypes.NewTxAssembleVinObject(vout.TxID, vout.OutIndex, vout.sequence, int64(lockTime), vout.InputType)
		if vinEncode == nil {
			return "", errors.New("input type not support:" + vout.InputType.String())
		}
		err := vinEncode.AssembleVin(mtx)
		if err != nil {
			return "", err
		}
		txtypes.InputTypeSet(i, vout.InputType)
	}

	for i := 0; i < len(outputs); i++ {
		o := outputs[i]
		voutEncode := txbasetypes.NewTxAssembleVoutObject(o.TargetAddress, o.Amount, o.TargetLockTime, o.OutputType)
		if voutEncode == nil {
			return "", errors.New("oupput type not support:" + o.OutputType.String())
		}
		err := voutEncode.AssembleVout(mtx)
		if err != nil {
			return "", err
		}
		txtypes.OutputTypeSet(i, o.OutputType)
	}
	mtxHex, err := mtx.Serialize()
	if err != nil {
		return "", err
	}
	typeIndex, err := txtypes.Encode()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mtxHex) + MTX_STR_SEPERATE + typeIndex, nil
}

func DecodePkString(pk string) (string, error) {
	b, err := txscript.PkStringToScript(pk)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func TxSign(privkeyStrs []string, rawTxStr string, network string) (string, error) {
	strArr := strings.Split(rawTxStr, MTX_STR_SEPERATE)
	rawTxStr = strArr[0]
	txtypeIndex := &TxTypeIndex{}
	if len(strArr) == 2 {
		txtypeIndex, _ = DecodeTxTypeIndex(strArr[1])
	}
	var param *params.Params
	switch network {
	case "mainnet":
		param = &params.MainNetParams
	case "testnet":
		param = &params.TestNetParams
	case "privnet":
		param = &params.PrivNetParams
	case "mixnet":
		param = &params.MixNetParams
	}

	if len(rawTxStr)%2 != 0 {
		return "", fmt.Errorf("invaild raw transaction : %s", rawTxStr)
	}
	serializedTx, err := hex.DecodeString(rawTxStr)
	if err != nil {
		return "", err
	}

	var redeemTx types.Transaction
	err = redeemTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return "", err
	}
	//
	for i := range redeemTx.TxIn {
		txSignBase := txbasetypes.NewTxSignObject(txtypeIndex.FindInputTxType(i))
		err = txSignBase.Sign(privkeyStrs[i], &redeemTx, i, param)
		if err != nil {
			return "", err
		}
	}

	mtxHex, err := marshal.MessageToHex(&redeemTx)
	if err != nil {
		return "", err
	}
	return mtxHex, nil
}

func TxDecode(network string, rawTxStr string) {
	var param *params.Params
	switch network {
	case "mainnet":
		param = &params.MainNetParams
	case "testnet":
		param = &params.TestNetParams
	case "privnet":
		param = &params.PrivNetParams
	case "mixnet":
		param = &params.MixNetParams
	}
	strArr := strings.Split(rawTxStr, MTX_STR_SEPERATE)
	rawTxStr = strArr[0]
	txTypeIndex := &TxTypeIndex{}
	if len(strArr) == 2 {
		txTypeIndex, _ = DecodeTxTypeIndex(strArr[1])
	}
	if len(rawTxStr)%2 != 0 {
		ErrExit(fmt.Errorf("invaild raw transaction : %s", rawTxStr))
	}
	serializedTx, err := hex.DecodeString(rawTxStr)
	if err != nil {
		ErrExit(err)
	}
	var tx types.Transaction
	err = tx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		ErrExit(err)
	}
	vins := marshal.MarshJsonVin(&tx)
	if len(strArr) == 2 {
		for i := range vins {
			vins[i].TxType = txTypeIndex.FindInputTxType(i).String()
		}
	}
	jsonTx := &json.OrderedResult{
		{Key: "txid", Val: tx.TxHash().String()},
		{Key: "txhash", Val: tx.TxHashFull().String()},
		{Key: "version", Val: int32(tx.Version)},
		{Key: "locktime", Val: tx.LockTime},
		{Key: "expire", Val: tx.Expire},
		{Key: "vin", Val: vins},
		{Key: "vout", Val: marshal.MarshJsonVout(&tx, nil, param)},
	}
	marshaledTx, err := jsonTx.MarshalJSON()
	if err != nil {
		ErrExit(err)
	}

	fmt.Printf("%s", marshaledTx)
}

func TxEncodeSTDO(version TxVersionFlag, lockTime TxLockTimeFlag, txIn TxInputsFlag, txOut TxOutputsFlag) {
	txInputs := []Input{}
	txOutputs := []Output{}
	for _, input := range txIn.inputs {
		txInputs = append(txInputs, Input{
			TxID:      hex.EncodeToString(input.txhash),
			OutIndex:  input.index,
			InputType: types.GetTxType(input.txtype),
			sequence:  input.sequence,
		})
	}
	for _, output := range txOut.outputs {
		atomic, err := types.NewAmount(output.amount)
		if err != nil {
			ErrExit(fmt.Errorf("fail to create the currency amount from a "+
				"floating point value %f : %w", output.amount, err))
		}
		txOutputs = append(txOutputs, Output{
			TargetAddress: output.target,
			OutputType:    types.GetTxType(output.txtype),
			Amount: types.Amount{
				Value: atomic.Value,
				Id:    types.CoinID(output.coinid),
			},
			TargetLockTime: int64(lockTime),
		})
	}
	mtxHex, err := TxEncode(uint32(version), uint32(lockTime), nil, txInputs, txOutputs)
	if err != nil {
		ErrExit(err)
	}
	fmt.Printf("%s\n", mtxHex)
}

func TxSignSTDO(privkeyStr string, rawTxStr string, network string) {
	mtxHex, err := TxSign([]string{privkeyStr}, rawTxStr, network)
	if err != nil {
		ErrExit(err)
	}
	fmt.Printf("%s\n", mtxHex)
}
