package qx

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/qx/scriptbasetypes"
	"log"
	"strconv"
	"strings"
	"time"
)

type Input struct {
	TxID       string
	OutIndex   uint32
	SignScript []byte
	sequence   uint32
	InputType  txscript.ScriptClass
	LockTime   int64
}

type Output struct {
	TargetAddress  string
	Amount         types.Amount
	TargetLockTime int64
	OutputType     txscript.ScriptClass
}

const MTX_STR_SEPERATE = "-"

func TxEncode(version uint32, lockTime uint32, timestamp *time.Time, inputs []Input, outputs []Output) (string, error) {
	mtx := types.NewTransaction()
	mtx.Version = version
	if lockTime != 0 {
		mtx.LockTime = lockTime
	}
	if timestamp != nil {
		mtx.Timestamp = *timestamp
	}

	txtypes := &ScriptTypeIndex{}
	for i, vin := range inputs {
		txtypes.InputTypeSet(i, vin.InputType, vin.LockTime)
		txHash, err := hash.NewHashFromStr(vin.TxID)
		if err != nil {
			log.Fatalln(err)
			return "", err
		}
		prevOut := types.NewOutPoint(txHash, vin.OutIndex)
		txIn := types.NewTxInput(prevOut, []byte{})
		if vin.sequence > 0 {
			txIn.Sequence = vin.sequence
		}
		//setting the locktime to 0, or
		//setting the locktime to be less than the current block height, or
		//setting the locktime to be less than the current time (but still above a threshold so that it is not confused for a block height), or
		//setting ALL txin sequence numbers to 0xffffffff.
		// check sequence and lockTime
		if vin.sequence == types.MaxTxInSequenceNum-1 && lockTime <= 0 {
			return "", errors.New("unlock cltvpubkeyhash script,locktime must > 0")
		}
		mtx.AddTxIn(txIn)
	}

	for i := 0; i < len(outputs); i++ {
		o := outputs[i]
		txtypes.OutputTypeSet(i, o.OutputType)
		addr, err := address.DecodeAddress(o.TargetAddress)
		if err != nil {
			return "", fmt.Errorf("could not decode "+
				"address: %v", err)
		}
		var pkScript []byte
		switch addr.(type) {
		case *address.PubKeyHashAddress:
		case *address.SecpPubKeyAddress:
		case *address.ScriptHashAddress:
		default:
			return "", fmt.Errorf("unsupport address type: %T", addr)
		}
		// if coinID is meerB the out address must be SecpPubKeyAddress
		if o.Amount.Id == types.MEERB {
			if _, ok := addr.(*address.SecpPubKeyAddress); !ok {
				return "", fmt.Errorf("out coinid is %v but the out address is: %v , not the SecpPubKeyAddress", o.Amount.Id, addr)
			}
		}
		switch o.OutputType {
		case txscript.CLTVPubKeyHashTy:
			if o.TargetLockTime <= 0 {
				return "", fmt.Errorf("can not set CLTVPubKeyHashTy ADDRESS:AMOUNT:COINID:SCRIPTTYPE:LOCKTIME")
			}
			if _, ok := addr.(*address.PubKeyHashAddress); !ok {
				return "", fmt.Errorf("locktype is %v but the out address is: %v , not the PubKeyHashAddress", o.OutputType.String(), addr)
			}
			pkScript, err = txscript.PayToCLTVPubKeyHashScript(addr.Script(), o.TargetLockTime)
			if err != nil {
				return "", err
			}
		case txscript.PubKeyTy:
			if _, ok := addr.(*address.SecpPubKeyAddress); !ok {
				return "", fmt.Errorf("locktype is %v but the out address is: %v , not the SecpPubKeyAddress", o.OutputType.String(), addr)
			}
			pkScript, err = txscript.PayToAddrScript(addr)
			if err != nil {
				return "", err
			}
		default: // pubkeyhash standard
			if _, ok := addr.(*address.PubKeyHashAddress); !ok {
				return "", fmt.Errorf("locktype is %v but the out address is: %v , not the PubKeyHashAddress", o.OutputType.String(), addr)
			}
			pkScript, err = txscript.PayToAddrScript(addr)
			if err != nil {
				return "", err
			}
		}
		txOut := types.NewTxOutput(o.Amount, pkScript)
		mtx.AddTxOut(txOut)
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

func TxSign(privkeyStrs []string, rawTxStr string, network string) (string, error) {
	strArr := strings.Split(rawTxStr, MTX_STR_SEPERATE)
	rawTxStr = strArr[0]
	txtypeIndex := &ScriptTypeIndex{}
	if len(strArr) == 2 {
		txtypeIndex, _ = DecodeScriptTypeIndex(strArr[1])
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
	if len(privkeyStrs) != len(redeemTx.TxIn) {
		return "", fmt.Errorf("vin length is %v , but private keys length is %v", len(redeemTx.TxIn), len(privkeyStrs))
	}
	for i := range redeemTx.TxIn {
		txSignBase := scriptbasetypes.NewTxSignObject(txtypeIndex.FindInputScriptType(i), txtypeIndex.FindInputScriptLockTime(i))
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
	txTypeIndex := &ScriptTypeIndex{}
	if len(strArr) == 2 {
		txTypeIndex, _ = DecodeScriptTypeIndex(strArr[1])
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
			vins[i].TxType = txTypeIndex.FindInputScriptType(i).String()
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
	var err error
	for _, input := range txIn.inputs {
		lockT := int64(0)
		if input.unlocktype == txscript.CLTVPubKeyHashTy.String() {
			lockT, err = strconv.ParseInt(input.args, 10, 46)
			if err != nil {
				ErrExit(fmt.Errorf("cltvpubkeyhash need a locktime or lockheight"))
			}
		}
		txInputs = append(txInputs, Input{
			TxID:      hex.EncodeToString(input.txhash),
			OutIndex:  input.index,
			InputType: scriptbasetypes.GetScriptType(input.unlocktype),
			sequence:  input.sequence,
			LockTime:  lockT,
		})
	}
	for _, output := range txOut.outputs {
		atomic, err := types.NewAmount(output.amount)
		if err != nil {
			ErrExit(fmt.Errorf("fail to create the currency amount from a "+
				"floating point value %f : %w", output.amount, err))
		}
		targetLock := int64(0)
		if output.locktype == txscript.CLTVPubKeyHashTy.String() {
			targetLock, err = strconv.ParseInt(output.args, 10, 46)
			if err != nil {
				ErrExit(fmt.Errorf("cltvpubkeyhash need a locktime or lockheight ADDRESS:AMOUNT:COINID:SCRIPTTYPE:LOCKTIME"))
			}
		}

		txOutputs = append(txOutputs, Output{
			TargetAddress: output.target,
			OutputType:    scriptbasetypes.GetScriptType(output.locktype),
			Amount: types.Amount{
				Value: atomic.Value,
				Id:    types.CoinID(output.coinid),
			},
			TargetLockTime: targetLock,
		})
	}
	mtxHex, err := TxEncode(uint32(version), uint32(lockTime), nil, txInputs, txOutputs)
	if err != nil {
		ErrExit(err)
	}
	fmt.Printf("%s\n", mtxHex)
}

func TxSignSTDO(privkeyStrs []string, rawTxStr string, network string) {
	mtxHex, err := TxSign(privkeyStrs, rawTxStr, network)
	if err != nil {
		ErrExit(err)
	}
	fmt.Printf("%s\n", mtxHex)
}
