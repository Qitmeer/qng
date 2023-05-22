/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	qtypes "github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/urfave/cli/v2"
	"math/big"
	"strconv"
	"strings"
	"time"
)

func ReverseBytes(bs *[]byte) *[]byte {
	length := len(*bs)
	for i := 0; i < length/2; i++ {
		index := length - 1 - i
		temp := (*bs)[index]
		(*bs)[index] = (*bs)[i]
		(*bs)[i] = temp
	}
	return bs
}

func NewMeerEVMAddress(pubkeyHex string) (common.Address, error) {
	pubkBytes, err := hex.DecodeString(pubkeyHex)
	if err != nil {
		return common.Address{}, err
	}

	publicKey, err := ecc.Secp256k1.ParsePubKey(pubkBytes)
	if err != nil {
		return common.Address{}, err
	}
	return crypto.PubkeyToAddress(*publicKey.ToECDSA()), nil
}

var (
	Precision = big.NewInt(params.Ether).Div(big.NewInt(params.Ether), big.NewInt(qtypes.AtomsPerCoin))
)

func CopyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

func TotalFees(block *types.Block, receipts []*types.Receipt) *big.Float {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Ether)))
}

func ToEVMHash(h *hash.Hash) common.Hash {
	ehb := h.Bytes()
	ReverseBytes(&ehb)
	return common.BytesToHash(ehb)
}

func FromEVMHash(h common.Hash) *hash.Hash {
	ehb := h.Bytes()
	ReverseBytes(&ehb)
	th, err := hash.NewHash(ehb)
	if err != nil {
		return nil
	}
	return th
}

func ToQNGTx(tx *types.Transaction, timestamp int64) *qtypes.Transaction {
	txmb, err := tx.MarshalBinary()
	if err != nil {
		log.Error(err.Error())
		return nil
	}

	qtxhb := tx.Hash().Bytes()
	ReverseBytes(&qtxhb)
	qtxh := hash.MustBytesToHash(qtxhb)

	mtx := qtypes.NewTransaction()

	if timestamp > 0 {
		mtx.Timestamp = time.Unix(timestamp, 0)
	}

	mtx.AddTxIn(&qtypes.TxInput{
		PreviousOut: *qtypes.NewOutPoint(&qtxh, qtypes.SupperPrevOutIndex),
		Sequence:    uint32(qtypes.TxTypeCrossChainVM),
		AmountIn:    qtypes.Amount{Id: qtypes.MEERB, Value: 0},
		SignScript:  txmb,
	})
	mtx.AddTxOut(&qtypes.TxOutput{
		Amount:   qtypes.Amount{Value: 0, Id: qtypes.MEERB},
		PkScript: opreturn.NewEVMTx().PKScript(),
	})

	return mtx
}

// Merge merges the given flag slices.
func Merge(groups ...[]cli.Flag) []cli.Flag {
	var ret []cli.Flag
	for _, group := range groups {
		ret = append(ret, group...)
	}
	return ret
}

func ProcessEnv(env string, identifier string, exclusionFlags []cli.Flag) ([]string, error) {
	result := []string{identifier}
	if len(env) <= 0 {
		return result, nil
	}
	// Detect unsupported flags
	if len(exclusionFlags) > 0 {
		for _, flag := range exclusionFlags {
			for _, name := range flag.Names() {
				if strings.Contains(env, name) {
					return nil, fmt.Errorf("%s does not support %s flag", identifier, name)
				}
			}
		}
	}
	if e, err := strconv.Unquote(env); err == nil {
		env = e
	}
	args := strings.Split(env, " ")
	if len(args) <= 0 {
		return result, nil
	}
	log.Debug(fmt.Sprintf("Initialize meerevm environment: %v %v ", len(args), args))
	result = append(result, args...)

	return result, nil
}

func DecodeTx(data []byte) (*types.Transaction,error) {
	if len(data) <= 2 {
		return nil,fmt.Errorf("No tx data:%v",data)
	}
	var txb []byte
	if data[0] == 48 && data[1] == 120 {
		txb = common.FromHex(string(data))
	}else{
		txb = data
	}
	var txmb = &types.Transaction{}
	if err := txmb.UnmarshalBinary(txb); err != nil {
		return nil, err
	}
	return txmb,nil
}

func ToTxHex(data []byte) []byte {
	if len(data) <= 2 {
		return nil
	}
	if data[0] == 48 && data[1] == 120 {
		return common.FromHex(string(data))
	}
	return data
}

func ToTxHexStr(data []byte) string {
	if len(data) <= 2 {
		return ""
	}
	if data[0] == 48 && data[1] == 120 {
		return string(data)
	}
	return hexutil.Encode(data)
}