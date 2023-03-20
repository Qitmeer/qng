package wallet

import (
	"encoding/hex"
	ejson "encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/qx"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/services/acct"
	"github.com/Qitmeer/qng/services/mempool"
	"time"
)

func (a *WalletManager) getAvailableUtxos(addr string, amount int64) ([]acct.UTXOResult, int64, error) {
	otxoList := make([]acct.UTXOResult, 0)
	utxos, err := a.am.GetUTXOs(addr)
	if err != nil {
		return nil, 0, err
	}
	sum := uint64(0)
	for _, utxo := range utxos {
		if utxo.Status != "unlocked" && utxo.Status != "valid" {
			continue
		}
		sum += utxo.Amount
		otxoList = append(otxoList, utxo)
		if sum > uint64(amount) {
			break
		}
	}
	return otxoList, int64(sum), err
}

func (a *WalletManager) sendTx(fromAddress string, amounts json.AddressAmountV3, targetLockTime, lockTime int64) (string, error) {
	amount := int64(0)
	outputs := make([]qx.Output, 0)
	for addres, v := range amounts {
		amount += v.Amount
		typ := txscript.PubkeyHashAltTy
		addr, err := address.DecodeAddress(addres)
		if err != nil {
			return "", err
		}
		switch addr.(type) {
		case *address.SecpPubKeyAddress:
			typ = txscript.PubKeyTy
		}
		outputs = append(outputs, qx.Output{
			TargetAddress:  addres,
			Amount:         types.Amount{v.Amount, types.CoinID(v.CoinId)},
			OutputType:     typ,
			TargetLockTime: targetLockTime,
		})
	}
	uxtoList, sum, err := a.getAvailableUtxos(fromAddress, amount)
	if err != nil {
		return "", err
	}
	if len(uxtoList) < 1 {
		return "", fmt.Errorf("%s balance not enough", fromAddress)
	}
	if sum <= amount {
		return "", fmt.Errorf("%s balance not enough , current:%d,need more than:%d", fromAddress, sum, amount)
	}
	//left := sum - amount.Value
	inputs := make([]qx.Input, 0)
	priKeyList := make([]string, 0)
	addr, _ := address.DecodeAddress(fromAddress)
	pri, ok := a.qks.unlocked[fromAddress]
	if !ok {
		return "", fmt.Errorf("please unlock %s first", fromAddress)
	}
	typ := txscript.PubKeyHashTy
	switch addr.(type) {
	case *address.SecpPubKeyAddress:
		typ = txscript.PubKeyTy
	default:
	}
	for _, utxo := range uxtoList {
		inputs = append(inputs, qx.Input{
			TxID:      utxo.PreTxHash,
			InputType: typ,
			OutIndex:  utxo.PreOutIdx})

		priKeyList = append(priKeyList, hex.EncodeToString(pri.PrivateKey.D.Bytes()))
	}
	timeNow := time.Now()

	raw, err := qx.TxEncode(1, uint32(lockTime), &timeNow, inputs, outputs)
	if err != nil {
		return "", err
	}
	signedRaw, err := qx.TxSign(priKeyList, raw, params.ActiveNetParams.Params.Name)
	if err != nil {
		return "", err
	}
	serializedTx, err := hex.DecodeString(signedRaw)
	if err != nil {
		return "", rpc.RpcDecodeHexError(signedRaw)
	}
	leftOutput := qx.Output{
		TargetAddress: fromAddress,
		Amount:        types.Amount{0, types.MEERA},
		OutputType:    typ,
	}
	b, _ := ejson.Marshal(leftOutput)
	serializedSize := len(serializedTx) + len(b)
	minFee := mempool.CalcMinRequiredTxRelayFee(int64(serializedSize),
		types.Amount{a.cfg.MinTxFee, types.MEERA})
	leftAmount := sum - amount - minFee
	for _, v := range inputs {
		fmt.Println(v.TxID)
		fmt.Println(v.OutIndex)
	}

	if leftAmount > 0 {
		leftOutput.Amount.Value = leftAmount
		outputs = append(outputs, leftOutput)
	}
	raw, err = qx.TxEncode(1, uint32(lockTime), &timeNow, inputs, outputs)
	if err != nil {
		return "", err
	}
	signedRaw, err = qx.TxSign(priKeyList, raw, params.ActiveNetParams.Params.Name)
	if err != nil {
		return "", err
	}
	serializedTx, err = hex.DecodeString(signedRaw)
	if err != nil {
		return "", rpc.RpcDecodeHexError(signedRaw)
	}
	return a.tm.ProcessRawTx(serializedTx, false)
}
