// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2016-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mining

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/merkle"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"time"
)

const (

	// kilobyte is the size of a kilobyte.
	// TODO, refactor the location of kilobyte const
	kilobyte = 1000

	// blockHeaderOverhead is the max number of bytes it takes to serialize
	// a block header and max possible transaction count.
	// TODO, refactor the location of blockHeaderOverhead const
	blockHeaderOverhead = types.MaxBlockHeaderPayload + s.MaxVarIntPayload
)

// coinbaseFlags is some extra data appended to the coinbase script
// sig.
type CoinbaseFlags string

const (
	CoinbaseFlagsStatic  CoinbaseFlags = "/qitmeer/"
	CoinbaseFlagsDynamic CoinbaseFlags = "/qng/"
)

// TxSource represents a source of transactions to consider for inclusion in
// new blocks.
//
// The interface contract requires that all of these methods are safe for
// concurrent access with respect to the source.
type TxSource interface {
	// LastUpdated returns the last time a transaction was added to or
	// removed from the source pool.
	LastUpdated() time.Time

	// MiningDescs returns a slice of mining descriptors for all the
	// transactions in the source pool.
	MiningDescs() []*types.TxDesc

	// HaveTransaction returns whether or not the passed transaction hash
	// exists in the source pool.
	HaveTransaction(hash *hash.Hash) bool

	// HaveAllTransactions returns whether or not all of the passed
	// transaction hashes exist in the source pool.
	HaveAllTransactions(hashes []hash.Hash) bool
}

// Allowed timestamp for a block building on the end of the provided best chain.
func MinimumMedianTime(bc *blockchain.BlockChain) time.Time {
	return bc.BestSnapshot().MedianTime.Add(time.Second)
}

// medianAdjustedTime returns the current time adjusted
func MedianAdjustedTime(bc *blockchain.BlockChain, timeSource model.MedianTimeSource) time.Time {
	newTimestamp := timeSource.AdjustedTime()
	minTimestamp := MinimumMedianTime(bc)
	if newTimestamp.Before(minTimestamp) {
		newTimestamp = minTimestamp
	}

	return newTimestamp
}

func StandardCoinbaseScript(nextBlockHeight uint64, extraNonce uint64, extraData string, flags CoinbaseFlags) ([]byte, error) {
	scriptBuilder := txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).AddData([]byte(flags))
	if len(extraData) > 0 {
		scriptBuilder = scriptBuilder.AddData([]byte(extraData))
	}
	return scriptBuilder.Script()
}

// standardCoinbaseOpReturn creates a standard OP_RETURN output to insert into
// coinbase to use as extranonces. The OP_RETURN pushes 32 bytes.
func standardCoinbaseOpReturn(enData []byte) ([]byte, error) {
	if len(enData) == 0 {
		return nil, nil
	}
	extraNonceScript, err := txscript.GenerateProvablyPruneableOut(enData)
	if err != nil {
		return nil, err
	}
	return extraNonceScript, nil
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
//
// See the comment for NewBlockTemplate for more information about why the nil
// address handling is useful.
func createCoinbaseTx(subsidyCache *blockchain.SubsidyCache, coinbaseScript []byte, bi *meerdag.BlueInfo, addr types.Address, params *params.Params, opReturnPkScript []byte) (*types.Tx, *types.TxOutput, *types.TxOutput, error) {
	tx := types.NewTransaction()
	tx.AddTxIn(&types.TxInput{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOut: *types.NewOutPoint(&hash.Hash{},
			types.MaxPrevOutIndex),
		Sequence:   types.MaxTxInSequenceNum,
		SignScript: coinbaseScript,
	})

	// Create a coinbase with correct block subsidy and extranonce.
	subsidy := blockchain.CalcBlockWorkSubsidy(subsidyCache,
		bi, params)
	tax := blockchain.CalcBlockTaxSubsidy(subsidyCache,
		bi, params)

	// output
	// Create the script to pay to the provided payment address if one was
	// specified.  Otherwise create a script that allows the coinbase to be
	// redeemable by anyone.
	var pksSubsidy []byte
	var err error
	if addr != nil {
		pksSubsidy, err = txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		scriptBuilder := txscript.NewScriptBuilder()
		pksSubsidy, err = scriptBuilder.AddOp(txscript.OP_TRUE).Script()
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if !params.HasTax() {
		subsidy += uint64(tax)
		tax = 0
	}
	// Subsidy paid to miner.
	tx.AddTxOut(&types.TxOutput{
		Amount:   types.Amount{Value: int64(subsidy), Id: types.MEERA},
		PkScript: pksSubsidy,
	})

	// Tax output.
	var taxOutput *types.TxOutput
	if params.HasTax() {
		taxOutput = &types.TxOutput{
			Amount:   types.Amount{Value: int64(tax), Id: types.MEERA},
			PkScript: params.OrganizationPkScript,
		}
	}

	// opReturnPkScript
	var opReturnOutput *types.TxOutput
	if len(opReturnPkScript) > 0 {
		opReturnOutput = &types.TxOutput{
			PkScript: opReturnPkScript,
		}
	} else {
		opReturnOutput = opreturn.GetOPReturnTxOutput(opreturn.NewLockAmount(int64(subsidy)))
	}

	return types.NewTx(tx), taxOutput, opReturnOutput, nil
}

func fillWitnessToCoinBase(blockTxns []*types.Tx) (*hash.Hash, error) {
	merkles := merkle.BuildMerkleTreeStore(blockTxns, true)
	txWitnessRoot := merkles[len(merkles)-1]
	witnessPreimage := append(txWitnessRoot.Bytes(), blockTxns[0].Tx.TxIn[0].SignScript...)
	witnessCommitment := hash.DoubleHashH(witnessPreimage[:])
	blockTxns[0].Tx.TxIn[0].PreviousOut.Hash = witnessCommitment
	blockTxns[0].RefreshHash()
	return txWitnessRoot, nil
}

func fillOutputsToCoinBase(coinbaseTx *types.Tx, blockFeesMap types.AmountMap, taxOutput *types.TxOutput, oprOutput *types.TxOutput) error {
	if len(coinbaseTx.Tx.TxOut) != blockchain.CoinbaseOutput_subsidy+1 {
		return fmt.Errorf("coinbase output error")
	}
	for k, v := range blockFeesMap {
		if v <= 0 || k == types.MEERA {
			continue
		}
		coinbaseTx.Tx.AddTxOut(&types.TxOutput{
			Amount:   types.Amount{Value: 0, Id: k},
			PkScript: coinbaseTx.Tx.TxOut[0].GetPkScript(),
		})
	}
	if taxOutput != nil {
		coinbaseTx.Tx.AddTxOut(taxOutput)
	}
	if oprOutput != nil {
		coinbaseTx.Tx.AddTxOut(oprOutput)
	}
	return nil
}

func IsSupportCoinbaseFlagsDynamic(coinbaseTx *types.Transaction) bool {
	if !coinbaseTx.IsCoinBase() {
		return false
	}

	ops, err := txscript.ParseScript(coinbaseTx.TxIn[0].SignScript)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if len(ops) < 3 {
		return false
	}

	cfd := CoinbaseFlags(ops[2].GetData())

	return cfd == CoinbaseFlagsDynamic
}

func DoCalculateTransactionsRoot(coinbaseTx *types.Transaction, merklePath []*hash.Hash, witnessRoot *hash.Hash, extraNonce uint64) (*hash.Hash, error) {
	if !IsSupportCoinbaseFlagsDynamic(coinbaseTx) {
		return nil, fmt.Errorf("Not support:%s\n", CoinbaseFlagsDynamic)
	}
	ops, _ := txscript.ParseScript(coinbaseTx.TxIn[0].SignScript)
	nextBlockHeight := txscript.GetInt64FromOpcode(ops[0])
	var extraData string
	if len(ops) >= 4 {
		extraData = string(ops[3].GetData())
	}
	coinbaseScript, err := StandardCoinbaseScript(uint64(nextBlockHeight), extraNonce, extraData, CoinbaseFlagsDynamic)
	if err != nil {
		return nil, err
	}
	if witnessRoot == nil {
		witnessRoot = &hash.ZeroHash
	}
	coinbaseTx.TxIn[0].SignScript = coinbaseScript
	witnessPreimage := append(witnessRoot.Bytes(), coinbaseTx.TxIn[0].SignScript...)
	witnessCommitment := hash.DoubleHashH(witnessPreimage[:])
	coinbaseTx.TxIn[0].PreviousOut.Hash = witnessCommitment

	coinbaseTxHash := coinbaseTx.TxHash()
	return merkle.CalculateMerkleTreeRootByPath(&coinbaseTxHash, merklePath), nil
}

func CalculateTransactionsRoot(coinbaseTx string, merklePath []string, witnessRoot string, extraNonce uint64) (*hash.Hash, error) {
	serializedTx, err := hex.DecodeString(coinbaseTx)
	if err != nil {
		return nil, err
	}
	var mtx types.Transaction
	err = mtx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, err
	}
	path := []*hash.Hash{}
	for _, mp := range merklePath {
		ph, err := hash.NewHashFromStr(mp)
		if err != nil {
			return nil, err
		}
		path = append(path, ph)
	}
	var wroot *hash.Hash
	if len(witnessRoot) > 0 {
		wr, err := hash.NewHashFromStr(witnessRoot)
		if err != nil {
			return nil, err
		}
		wroot = wr
	}

	return DoCalculateTransactionsRoot(&mtx, path, wroot, extraNonce)
}
