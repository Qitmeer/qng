package tx

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/common/math"
	qconsensus "github.com/Qitmeer/qng/consensus/vm"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain/token"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/json"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/crypto/ecc"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"strconv"
	"strings"
	"time"
)

func (tm *TxManager) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicTxAPI(tm),
			Public:    true,
		},
		{
			NameSpace: cmds.TestNameSpace,
			Service:   NewPrivateTxAPI(tm),
			Public:    false,
		},
		tm.txMemPool.API(),
	}
}

type PublicTxAPI struct {
	txManager *TxManager
}

func NewPublicTxAPI(tm *TxManager) *PublicTxAPI {
	ptapi := PublicTxAPI{tm}
	return &ptapi
}

func (api *PublicTxAPI) CreateRawTransaction(inputs []json.TransactionInput, amounts json.Amounts, lockTime *int64) (interface{}, error) {
	aa := json.AdreesAmount{}
	if len(amounts) > 0 {
		for k, v := range amounts {
			aa[k] = json.Amout{CoinId: uint16(types.MEERA), Amount: int64(v)}
		}
	}
	return api.txManager.CreateRawTransactionV2(inputs, aa, lockTime)
}

func (api *PublicTxAPI) DecodeRawTransaction(hexTx string) (interface{}, error) {
	// Deserialize the transaction.
	hexStr := hexTx
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, rpc.RpcDecodeHexError(hexStr)
	}
	var mtx types.Transaction
	err = mtx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, rpc.RpcDeserializationError("Could not decode Tx: %v",
			err)
	}

	log.Trace("decodeRawTx", "hex", hexStr)
	log.Trace("decodeRawTx", "hex", serializedTx)

	// Create and return the result.
	txReply := &json.OrderedResult{
		{Key: "txid", Val: mtx.TxHash().String()},
		{Key: "txhash", Val: mtx.TxHashFull().String()},
		{Key: "version", Val: int32(mtx.Version)},
		{Key: "locktime", Val: mtx.LockTime},
		{Key: "timestamp", Val: mtx.Timestamp.Format(time.RFC3339)},
		{Key: "vin", Val: marshal.MarshJsonVin(&mtx)},
		{Key: "vout", Val: marshal.MarshJsonVout(&mtx, nil, params.ActiveNetParams.Params)},
	}
	return txReply, nil
}

func (api *PublicTxAPI) SendRawTransaction(hexTx string, allowHighFees *bool) (interface{}, error) {
	hexStr := hexTx
	highFees := false
	if allowHighFees != nil {
		highFees = *allowHighFees
	}
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	serializedTx, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", rpc.RpcDecodeHexError(hexStr)
	}
	return api.txManager.ProcessRawTx(serializedTx, highFees)
}

func (api *PublicTxAPI) GetRawTransaction(txHash hash.Hash, verbose bool) (interface{}, error) {

	var mtx *types.Tx
	var blkHash *hash.Hash
	//var blkOrder uint64
	var blkHashStr string
	var confirmations int64

	// Try to fetch the transaction from the memory pool and if that fails,
	// try the block database.
	tx, _ := api.txManager.txMemPool.FetchTransaction(&txHash)

	if tx == nil {
		//not found from mem-pool, try db
		txIndex := api.txManager.indexManager.TxIndex()
		if txIndex == nil {
			return nil, fmt.Errorf("the transaction index " +
				"must be enabled to query the blockchain (specify --txindex in configuration)")
		}
		// Look up the location of the transaction.
		var blockRegion *database.BlockRegion
		var err error

		blockRegion, err = txIndex.TxBlockRegion(txHash)
		if err != nil {
			return nil, errors.New("Failed to retrieve transaction location")
		}
		var dtx *types.Transaction
		if blockRegion == nil {
			if api.txManager.indexManager.InvalidTxIndex() != nil {
				dtx, err = api.txManager.indexManager.InvalidTxIndex().Get(&txHash)
				if err != nil {
					return nil, errors.New("Failed to retrieve transaction location")
				}

				if !verbose {
					hexStr, err := marshal.MessageToHex(dtx)
					if err != nil {
						return nil, err
					}
					return hexStr, nil
				}
			} else {
				return nil, rpc.RpcNoTxInfoError(&txHash)
			}
		} else {

			// Load the raw transaction bytes from the database.
			var txBytes []byte
			err = api.txManager.db.View(func(dbTx database.Tx) error {
				var err error
				txBytes, err = dbTx.FetchBlockRegion(blockRegion)
				return err
			})
			if err != nil {
				return nil, rpc.RpcNoTxInfoError(&txHash)
			}

			// When the verbose flag isn't set, simply return the serialized
			// transaction as a hex-encoded string.  This is done here to
			// avoid deserializing it only to reserialize it again later.
			if !verbose {
				return hex.EncodeToString(txBytes), nil
			}

			// Grab the block height.
			blkHash = blockRegion.Hash
			/*blkOrder, err = api.txManager.bm.GetChain().BlockOrderByHash(blkHash)
			if err != nil {
				context := "Failed to retrieve block height"
				return nil, rpc.RpcInternalError(err.Error(), context)
			}*/

			// Deserialize the transaction
			var msgTx types.Transaction
			err = msgTx.Deserialize(bytes.NewReader(txBytes))
			log.Trace("GetRawTx", "hex", hex.EncodeToString(txBytes))
			if err != nil {
				context := "Failed to deserialize transaction"
				return nil, rpc.RpcInternalError(err.Error(), context)
			}
			dtx = &msgTx
		}

		mtx = types.NewTx(dtx)
		mtx.IsDuplicate = api.txManager.GetChain().IsDuplicateTx(mtx.Hash(), blkHash)
	} else {
		// When the verbose flag isn't set, simply return the
		// network-serialized transaction as a hex-encoded string.
		if !verbose {
			// Note that this is intentionally not directly
			// returning because the first return value is a
			// string and it would result in returning an empty
			// string to the client instead of nothing (nil) in the
			// case of an error.
			hexStr, err := marshal.MessageToHex(tx.Transaction())
			if err != nil {
				return nil, err
			}

			return hexStr, nil
		}

		mtx = tx
	}
	txsvalid := true
	coinbaseAmout := types.AmountMap{}
	if blkHash != nil {
		blkHashStr = blkHash.String()
		ib := api.txManager.GetChain().BlockDAG().GetBlock(blkHash)
		if ib != nil {
			confirmations = int64(api.txManager.GetChain().BlockDAG().GetConfirmations(ib.GetID()))
			txsvalid = !ib.GetState().GetStatus().KnownInvalid()
		}

		if mtx.Tx.IsCoinBase() {
			coinbaseFees := api.txManager.GetChain().GetFees(blkHash)
			if coinbaseFees == nil {
				coinbaseAmout[mtx.Tx.TxOut[0].Amount.Id] = mtx.Tx.TxOut[0].Amount.Value
			} else {
				coinbaseAmout = coinbaseFees
				coinbaseAmout[mtx.Tx.TxOut[0].Amount.Id] += mtx.Tx.TxOut[0].Amount.Value
			}
		}
	}
	if tx != nil {
		confirmations = 0
	}
	return marshal.MarshalJsonTransaction(mtx, api.txManager.consensus.Params(), blkHashStr, confirmations, coinbaseAmout, txsvalid)
}

// Returns information about an unspent transaction output
// 1. txid           (string, required)                The hash of the transaction
// 2. vout           (numeric, required)               The index of the output
// 3. includemempool (boolean, optional, default=true) Include the mempool when true
//
// Result:
// {
// "bestblock": "value",        (string)          The block hash that contains the transaction output
// "confirmations": n,          (numeric)         The number of confirmations
// "amount": n.nnn,             (numeric)         The transaction amount
// "scriptPubKey": {            (object)          The public key script used to pay coins as a JSON object
//
//	 "asm": "value",             (string)          Disassembly of the script
//	 "hex": "value",             (string)          Hex-encoded bytes of the script
//	 "reqSigs": n,               (numeric)         The number of required signatures
//	 "type": "value",            (string)          The type of the script (e.g. 'pubkeyhash')
//	 "addresses": ["value",...], (array of string) The qitmeer addresses associated with this script
//	},
//
// "coinbase": true|false,      (boolean)         Whether or not the transaction is a coinbase
// }
func (api *PublicTxAPI) GetUtxo(txHash hash.Hash, vout uint32, includeMempool *bool) (interface{}, error) {

	// If requested and the tx is available in the mempool try to fetch it
	// from there, otherwise attempt to fetch from the block database.
	var bestBlockHash string
	var confirmations int64
	var txVersion uint32
	var amount types.Amount
	var pkScript []byte
	var isCoinbase bool

	// by default try to search mempool tx
	includeMempoolTx := true
	if includeMempool != nil {
		includeMempoolTx = *includeMempool
	}

	// try mempool by default
	if includeMempoolTx {
		txFromMempool, _ := api.txManager.txMemPool.FetchTransaction(&txHash)
		if txFromMempool != nil {
			tx := txFromMempool.Transaction()
			txOut := tx.TxOut[vout]
			if txOut == nil {
				return nil, nil
			}
			best := api.txManager.GetChain().BestSnapshot()
			bestBlockHash = best.Hash.String()
			confirmations = 0
			txVersion = tx.Version
			amount = txOut.Amount
			pkScript = txOut.PkScript
			isCoinbase = tx.IsCoinBase()
		}
	}

	// otherwise try to lookup utxo set
	if bestBlockHash == "" {
		out := types.TxOutPoint{Hash: txHash, OutIndex: vout}
		entry, err := api.txManager.GetChain().FetchUtxoEntry(out)
		if err != nil {
			return nil, rpc.RpcNoTxInfoError(&txHash)
		}
		if entry == nil || entry.IsSpent() {
			return nil, nil
		}
		best := api.txManager.GetChain().BestSnapshot()
		bestBlockHash = best.Hash.String()

		amount = entry.Amount()
		if hash.ZeroHash.IsEqual(entry.BlockHash()) {
			confirmations = 0
		} else {
			block := api.txManager.GetChain().BlockDAG().GetBlock(entry.BlockHash())
			if block == nil {
				confirmations = 0
			} else {
				confirmations = int64(best.GraphState.GetLayer() - block.GetLayer())
			}
			if entry.IsCoinBase() {
				//TODO, even the entry is coinbase, should not change the amount by tx fee, need consider output index
				amount.Value += api.txManager.GetChain().GetFeeByCoinID(block.GetHash(), amount.Id)
			}
		}

		pkScript = entry.PkScript()
		isCoinbase = entry.IsCoinBase()
	}

	// Disassemble script into single line printable format.  The
	// disassembled string will contain [error] inline if the script
	// doesn't fully parse, so ignore the error here.
	script := pkScript
	disbuf, _ := txscript.DisasmString(script)

	// Get further info about the script.  Ignore the error here since an
	// error means the script couldn't parse and there is no additional
	// information about it anyways.
	scriptClass, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(script, api.txManager.consensus.Params())
	addresses := make([]string, len(addrs))
	for i, addr := range addrs {
		addresses[i] = addr.String()
	}
	txOutReply := &json.GetUtxoResult{
		BestBlock:     bestBlockHash,
		Confirmations: confirmations,
		CoinId:        uint16(amount.Id),
		Amount:        amount.ToUnit(types.AmountCoin),
		Version:       int32(txVersion),
		ScriptPubKey: json.ScriptPubKeyResult{
			Asm:       disbuf,
			Hex:       hex.EncodeToString(pkScript),
			ReqSigs:   int32(reqSigs),
			Type:      scriptClass.String(),
			Addresses: addresses,
		},
		Coinbase: isCoinbase,
	}
	return txOutReply, nil
}

// handleSearchRawTransactions implements the searchrawtransactions command.
func (api *PublicTxAPI) GetRawTransactions(addre string, vinext *bool, count *uint, skip *uint, revers *bool, verbose *bool, filterAddrs *[]string) (interface{}, error) {
	addrIndex := api.txManager.indexManager.AddrIndex()
	if addrIndex == nil {
		return nil, fmt.Errorf("Address index must be enabled (--addrindex)")
	}
	vinExtra := false
	if vinext != nil {
		vinExtra = *vinext
	}

	if vinExtra && api.txManager.indexManager.TxIndex() == nil {
		return nil, fmt.Errorf("Transaction index must be enabled (--txindex)")
	}
	params := api.txManager.consensus.Params()
	addr, err := address.DecodeAddress(addre)
	if err != nil {
		return nil, fmt.Errorf("Invalid address or key: " + err.Error())
	}
	numRequested := uint(100)
	if count != nil {
		numRequested = *count
	}
	if numRequested == 0 {
		return nil, nil
	}

	var numToSkip uint
	if skip != nil {
		numToSkip = *skip
	}

	var reverse bool
	if revers != nil {
		reverse = *revers
	}

	//
	numSkipped := uint32(0)
	addressTxns := make([]retrievedTx, 0, numRequested)
	if reverse {
		mpTxns, mpSkipped := api.fetchMempoolTxnsForAddress(addr,
			uint32(numToSkip), uint32(numRequested))
		numSkipped += mpSkipped
		for _, tx := range mpTxns {
			addressTxns = append(addressTxns, retrievedTx{tx: tx})
		}
	}

	// Fetch transactions from the database in the desired order if more are
	// needed.
	if uint(len(addressTxns)) < numRequested {
		err = api.txManager.db.View(func(dbTx database.Tx) error {
			regions, dbSkipped, err := addrIndex.TxRegionsForAddress(
				dbTx, addr, uint32(numToSkip)-numSkipped,
				uint32(numRequested-uint(len(addressTxns))), reverse)
			if err != nil {
				return err
			}

			// Load the raw transaction bytes from the database.
			serializedTxns, err := dbTx.FetchBlockRegions(regions)
			if err != nil {
				return err
			}

			// Add the transaction and the hash of the block it is
			// contained in to the list.  Note that the transaction
			// is left serialized here since the caller might have
			// requested non-verbose output and hence there would be
			// no point in deserializing it just to reserialize it
			// later.
			for i, serializedTx := range serializedTxns {
				addressTxns = append(addressTxns, retrievedTx{
					txBytes: serializedTx,
					blkHash: regions[i].Hash,
				})
			}
			numSkipped += dbSkipped

			return nil
		})
		if err != nil {
			context := "Failed to load address index entries"
			return nil, fmt.Errorf("%s %s", err.Error(), context)
		}

	}

	// Add transactions from mempool last if client did not request reverse
	// order and the number of results is still under the number requested.
	if !reverse && uint(len(addressTxns)) < numRequested {
		// Transactions in the mempool are not in a block header yet,
		// so the block header field in the retieved transaction struct
		// is left nil.
		mpTxns, mpSkipped := api.fetchMempoolTxnsForAddress(addr,
			uint32(numToSkip)-numSkipped, uint32(numRequested-
				uint(len(addressTxns))))
		numSkipped += mpSkipped
		for _, tx := range mpTxns {
			addressTxns = append(addressTxns, retrievedTx{tx: tx})
		}
	}

	// Address has never been used if neither source yielded any results.
	if len(addressTxns) == 0 {
		return nil, fmt.Errorf("No information available about address")
	}

	// Serialize all of the transactions to hex.
	hexTxns := make([]string, len(addressTxns))
	for i := range addressTxns {
		// Simply encode the raw bytes to hex when the retrieved
		// transaction is already in serialized form.
		rtx := &addressTxns[i]
		if rtx.txBytes != nil {
			hexTxns[i] = hex.EncodeToString(rtx.txBytes)
			continue
		}

		// Serialize the transaction first and convert to hex when the
		// retrieved transaction is the deserialized structure.
		hexTxns[i], err = marshal.MessageToHex(rtx.tx.Tx)
		if err != nil {
			return nil, err
		}
	}

	// When not in verbose mode, simply return a list of serialized txns.
	if verbose != nil && !(*verbose) {
		return hexTxns, nil
	}

	// Normalize the provided filter addresses (if any) to ensure there are
	// no duplicates.
	filterAddrMap := make(map[string]struct{})
	if filterAddrs != nil && len(*filterAddrs) > 0 {
		for _, addr := range *filterAddrs {
			filterAddrMap[addr] = struct{}{}
		}
	}

	// The verbose flag is set, so generate the JSON object and return it.
	srtList := make([]json.GetRawTransactionsResult, len(addressTxns))
	for i := range addressTxns {
		// The deserialized transaction is needed, so deserialize the
		// retrieved transaction if it's in serialized form (which will
		// be the case when it was lookup up from the database).
		// Otherwise, use the existing deserialized transaction.
		rtx := &addressTxns[i]
		var mtx *types.Tx
		if rtx.tx == nil {
			// Deserialize the transaction.

			mtxTx := &types.Transaction{}
			err := mtxTx.Deserialize(bytes.NewReader(rtx.txBytes))
			if err != nil {
				context := "Failed to deserialize transaction"
				return nil, fmt.Errorf("%s %s", err.Error(), context)
			}
			mtx = types.NewTx(mtxTx)
		} else {
			mtx = types.NewTx(rtx.tx.Tx)
		}

		result := &srtList[i]
		result.Hex = hexTxns[i]
		result.Txid = mtx.Tx.TxHash().String()
		result.Vin, err = api.createVinListPrevOut(mtx, params, vinExtra,
			filterAddrMap)
		if err != nil {
			return nil, err
		}

		if mtx.Tx.IsCoinBase() {
			amountMap := api.txManager.GetChain().GetFees(rtx.blkHash)
			result.Vout = marshal.MarshJsonCoinbaseVout(mtx.Tx, filterAddrMap, params, amountMap)
		} else {
			result.Vout = marshal.MarshJsonVout(mtx.Tx, filterAddrMap, params)
		}
		result.Version = mtx.Tx.Version
		result.LockTime = mtx.Tx.LockTime

		// Transactions grabbed from the mempool aren't yet in a block,
		// so conditionally fetch block details here.  This will be
		// reflected in the final JSON output (mempool won't have
		// confirmations or block information).
		var blkHeader *types.BlockHeader
		var blkHashStr string
		if blkHash := rtx.blkHash; blkHash != nil {
			// Fetch the header from chain.
			header, err := api.txManager.GetChain().HeaderByHash(blkHash)
			if err != nil {
				return nil, rpc.RpcInternalError("Block not found", "")
			}
			blkHeader = &header
			blkHashStr = blkHash.String()
		}

		// Add the block information to the result if there is any.
		if blkHeader != nil {
			// This is not a typo, they are identical in Bitcoin
			// Core as well.
			result.Time = blkHeader.Timestamp.Unix()
			result.Blocktime = blkHeader.Timestamp.Unix()
			result.BlockHash = blkHashStr
			result.Confirmations = uint64(api.txManager.GetChain().BlockDAG().GetConfirmations(
				api.txManager.GetChain().BlockDAG().GetBlockId(rtx.blkHash)))
		}
	}

	return srtList, nil
}

func (api *PublicTxAPI) fetchMempoolTxnsForAddress(addr types.Address, numToSkip, numRequested uint32) ([]*types.Tx, uint32) {
	// There are no entries to return when there are less available than the
	// number being skipped.
	mpTxns := api.txManager.indexManager.AddrIndex().UnconfirmedTxnsForAddress(addr)
	numAvailable := uint32(len(mpTxns))
	if numToSkip > numAvailable {
		return nil, numAvailable
	}

	// Filter the available entries based on the number to skip and number
	// requested.
	rangeEnd := numToSkip + numRequested
	if rangeEnd > numAvailable {
		rangeEnd = numAvailable
	}
	return mpTxns[numToSkip:rangeEnd], numToSkip
}

type retrievedTx struct {
	txBytes []byte
	blkHash *hash.Hash // Only set when transaction is in a block.
	tx      *types.Tx
}

func (api *PublicTxAPI) createVinListPrevOut(mtx *types.Tx, chainParams *params.Params, vinExtra bool, filterAddrMap map[string]struct{}) ([]json.VinPrevOut, error) {
	// Coinbase transactions only have a single txin by definition.
	if mtx.Tx.IsCoinBase() {
		// Only include the transaction if the filter map is empty
		// because a coinbase input has no addresses and so would never
		// match a non-empty filter.
		if len(filterAddrMap) != 0 {
			return nil, nil
		}

		txIn := mtx.Tx.TxIn[0]
		vinList := make([]json.VinPrevOut, 1)
		vinList[0].Coinbase = hex.EncodeToString(txIn.SignScript)
		vinList[0].Sequence = txIn.Sequence
		return vinList, nil
	}

	// Use a dynamically sized list to accommodate the address filter.
	vinList := make([]json.VinPrevOut, 0, len(mtx.Tx.TxIn))

	// Lookup all of the referenced transaction outputs needed to populate
	// the previous output information if requested.
	var originOutputs map[types.TxOutPoint]types.TxOutput
	if vinExtra || len(filterAddrMap) > 0 {
		var err error
		originOutputs, err = api.fetchInputTxos(mtx)
		if err != nil {
			return nil, err
		}
	}

	for _, txIn := range mtx.Tx.TxIn {
		// The disassembled string will contain [error] inline
		// if the script doesn't fully parse, so ignore the
		// error here.
		disbuf, _ := txscript.DisasmString(txIn.SignScript)

		// Create the basic input entry without the additional optional
		// previous output details which will be added later if
		// requested and available.
		prevOut := &txIn.PreviousOut
		vinEntry := json.VinPrevOut{
			Txid:     prevOut.Hash.String(),
			Vout:     prevOut.OutIndex,
			Sequence: txIn.Sequence,
			ScriptSig: &json.ScriptSig{
				Asm: disbuf,
				Hex: hex.EncodeToString(txIn.SignScript),
			},
		}

		// Add the entry to the list now if it already passed the filter
		// since the previous output might not be available.
		passesFilter := len(filterAddrMap) == 0
		if passesFilter {
			vinList = append(vinList, vinEntry)
		}

		// Only populate previous output information if requested and
		// available.
		if len(originOutputs) == 0 {
			continue
		}
		originTxOut, ok := originOutputs[*prevOut]
		if !ok {
			continue
		}

		// Ignore the error here since an error means the script
		// couldn't parse and there is no additional information about
		// it anyways.
		_, addrs, _, _ := txscript.ExtractPkScriptAddrs(originTxOut.PkScript, chainParams)

		// Encode the addresses while checking if the address passes the
		// filter when needed.
		encodedAddrs := make([]string, len(addrs))
		for j, addr := range addrs {
			encodedAddr := addr.String()
			encodedAddrs[j] = encodedAddr

			// No need to check the map again if the filter already
			// passes.
			if passesFilter {
				continue
			}
			if _, exists := filterAddrMap[encodedAddr]; exists {
				passesFilter = true
			}
		}

		// Ignore the entry if it doesn't pass the filter.
		if !passesFilter {
			continue
		}

		// Add entry to the list if it wasn't already done above.
		if len(filterAddrMap) != 0 {
			vinList = append(vinList, vinEntry)
		}

		// Update the entry with previous output information if
		// requested.
		if vinExtra {
			vinListEntry := &vinList[len(vinList)-1]
			vinListEntry.PrevOut = &json.PrevOut{
				Addresses: encodedAddrs,
				CoinId:    uint16(originTxOut.Amount.Id),
				Value:     originTxOut.Amount.ToCoin(),
			}
		}
	}

	return vinList, nil
}

func (api *PublicTxAPI) fetchInputTxos(tx *types.Tx) (map[types.TxOutPoint]types.TxOutput, error) {
	mp := api.txManager.txMemPool
	originOutputs := make(map[types.TxOutPoint]types.TxOutput)
	for txInIndex, txIn := range tx.Tx.TxIn {
		// Attempt to fetch and use the referenced transaction from the
		// memory pool.
		origin := &txIn.PreviousOut
		originTx, err := mp.FetchTransaction(&origin.Hash)
		if err == nil {
			txOuts := originTx.Tx.TxOut
			if origin.OutIndex >= uint32(len(txOuts)) {
				return nil, fmt.Errorf("unable to find output "+
					"%v referenced from transaction %s:%d",
					origin, tx.Tx.TxHash(), txInIndex)
			}

			originOutputs[*origin] = *txOuts[origin.OutIndex]
			continue
		}

		// Look up the location of the transaction.
		blockRegion, err := api.txManager.indexManager.TxIndex().TxBlockRegion(origin.Hash)
		if err != nil {
			context := "Failed to retrieve transaction location"
			return nil, rpc.RpcInternalError(err.Error(), context)
		}
		if blockRegion == nil {
			return nil, rpc.RpcNoTxInfoError(&origin.Hash)
		}

		// Load the raw transaction bytes from the database.
		var txBytes []byte
		err = api.txManager.db.View(func(dbTx database.Tx) error {
			var err error
			txBytes, err = dbTx.FetchBlockRegion(blockRegion)
			return err
		})
		if err != nil {
			return nil, rpc.RpcNoTxInfoError(&origin.Hash)
		}

		// Deserialize the transaction
		msgTx := &types.Transaction{}
		err = msgTx.Deserialize(bytes.NewReader(txBytes))
		if err != nil {
			context := "Failed to deserialize transaction"
			return nil, rpc.RpcInternalError(err.Error(), context)
		}

		// Add the referenced output to the map.
		if origin.OutIndex >= uint32(len(msgTx.TxOut)) {
			errStr := fmt.Sprintf("unable to find output %v "+
				"referenced from transaction %s:%d", origin,
				tx.Tx.TxHash(), txInIndex)
			return nil, rpc.RpcInternalError(errStr, "")
		}
		originOutputs[*origin] = *msgTx.TxOut[origin.OutIndex]
	}

	return originOutputs, nil
}

func (api *PublicTxAPI) GetRawTransactionByHash(txHash hash.Hash, verbose bool) (interface{}, error) {
	txIndex := api.txManager.indexManager.TxIndex()
	if txIndex == nil {
		return nil, fmt.Errorf("the transaction index " +
			"must be enabled to query the blockchain (specify --txindex in configuration)")
	}
	var txid *hash.Hash
	var err error
	txid, err = txIndex.GetTxIdByHash(txHash)
	if err != nil {
		if api.txManager.indexManager.InvalidTxIndex() != nil {
			txid, err = api.txManager.indexManager.InvalidTxIndex().GetIdByHash(&txHash)
			if err != nil {
				return nil, fmt.Errorf("no tx")
			}
		} else {
			return nil, fmt.Errorf("no tx")
		}
	}
	return api.GetRawTransaction(*txid, verbose)
}

func (api *PublicTxAPI) GetMeerEVMTxHashByID(txid hash.Hash) (interface{}, error) {
	var mtx *types.Tx
	tx, _ := api.txManager.txMemPool.FetchTransaction(&txid)
	if tx == nil {
		txIndex := api.txManager.indexManager.TxIndex()
		if txIndex == nil {
			return nil, fmt.Errorf("the transaction index " +
				"must be enabled to query the blockchain (specify --txindex in configuration)")
		}
		var blockRegion *database.BlockRegion
		var err error

		blockRegion, err = txIndex.TxBlockRegion(txid)
		if err != nil {
			return nil, errors.New("Failed to retrieve transaction location")
		}
		if blockRegion == nil {
			if api.txManager.indexManager.InvalidTxIndex() != nil {
				dtx, err := api.txManager.indexManager.InvalidTxIndex().Get(&txid)
				if err != nil {
					return nil, errors.New("Failed to retrieve transaction location")
				}
				mtx = types.NewTx(dtx)
			} else {
				return nil, rpc.RpcNoTxInfoError(&txid)
			}
		} else {
			var txBytes []byte
			err = api.txManager.db.View(func(dbTx database.Tx) error {
				var err error
				txBytes, err = dbTx.FetchBlockRegion(blockRegion)
				return err
			})
			if err != nil {
				return nil, rpc.RpcNoTxInfoError(&txid)
			}
			var msgTx types.Transaction
			err = msgTx.Deserialize(bytes.NewReader(txBytes))
			if err != nil {
				context := "Failed to deserialize transaction"
				return nil, rpc.RpcInternalError(err.Error(), context)
			}
			mtx = types.NewTx(&msgTx)
		}
	} else {
		mtx = tx
	}
	if !types.IsCrossChainVMTx(mtx.Tx) {
		return nil, fmt.Errorf("%s is not %v", txid, types.DetermineTxType(mtx.Tx))
	}
	return fmt.Sprintf("0x%s", mtx.Tx.TxIn[0].PreviousOut.Hash.String()), nil
}

func (api *PublicTxAPI) GetTxIDByMeerEVMTxHash(etxh hash.Hash) (interface{}, error) {
	vmi := api.txManager.GetChain().VMService()
	etxs, txhs, err := vmi.GetTxsFromMempool()
	if err != nil {
		return nil, err
	}
	if len(txhs) > 0 {
		for i := 0; i < len(txhs); i++ {
			if txhs[i].IsEqual(&etxh) {
				return etxs[i].TxHash().String(), nil
			}
		}
	}

	bid := vmi.GetBlockIDByTxHash(&etxh)
	if bid == 0 {
		return nil, fmt.Errorf("No meerevm tx:%s", etxh.String())
	}
	b := api.txManager.GetChain().GetBlockByNumber(bid)
	if b == nil {
		return nil, fmt.Errorf("Can't find block: number=%d  evm tx hash=%s", bid, etxh.String())
	}
	block, err := api.txManager.GetChain().FetchBlockByHash(b.GetHash())
	if err != nil {
		return nil, err
	}
	for _, tx := range block.Transactions() {
		if types.IsCrossChainVMTx(tx.Tx) {
			if etxh.IsEqual(&tx.Tx.TxIn[0].PreviousOut.Hash) {
				return tx.Hash().String(), nil
			}
		}
	}
	return nil, fmt.Errorf("No meerevm tx:%s", etxh.String())
}

type PrivateTxAPI struct {
	txManager *TxManager
}

func NewPrivateTxAPI(tm *TxManager) *PrivateTxAPI {
	ptapi := PrivateTxAPI{tm}
	return &ptapi
}

func (api *PrivateTxAPI) TxSign(privkeyStr string, rawTxStr string, tokenPrivkeyStr *string) (interface{}, error) {
	privkeyByte, err := hex.DecodeString(privkeyStr)
	if err != nil {
		return nil, err
	}
	if len(privkeyByte) != 32 {
		return nil, fmt.Errorf("error:%d", len(privkeyByte))
	}
	privateKey, _ := ecc.Secp256k1.PrivKeyFromBytes(privkeyByte)
	param := params.ActiveNetParams.Params

	if len(rawTxStr)%2 != 0 {
		return nil, fmt.Errorf("rawTxStr:%d", len(rawTxStr))
	}

	serializedTx, err := hex.DecodeString(rawTxStr)
	if err != nil {
		return nil, err
	}

	var redeemTx types.Transaction
	err = redeemTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, err
	}
	var kdb txscript.KeyClosure = func(types.Address) (ecc.PrivateKey, bool, error) {
		return privateKey, true, nil // compressed is true
	}
	//
	if types.IsTokenNewTx(&redeemTx) ||
		types.IsTokenRenewTx(&redeemTx) ||
		types.IsTokenValidateTx(&redeemTx) ||
		types.IsTokenInvalidateTx(&redeemTx) {
		if len(param.TokenAdminPkScript) <= 0 {
			return nil, fmt.Errorf("No token admin pk script.\n")
		}
		sigScript, err := txscript.SignTxOutput(param, &redeemTx, 0, param.TokenAdminPkScript, txscript.SigHashAll, kdb, nil, nil, ecc.ECDSA_Secp256k1)
		if err != nil {
			return nil, err
		}
		redeemTx.TxIn[0].SignScript = sigScript
	} else if types.IsCrossChainImportTx(&redeemTx) {
		itx, err := qconsensus.NewImportTx(&redeemTx)
		if err != nil {
			return nil, err
		}
		err = itx.Sign(privateKey)
		if err != nil {
			return nil, err
		}
	} else {
		txIndex := api.txManager.indexManager.TxIndex()
		if txIndex == nil {
			return nil, fmt.Errorf("the transaction index " +
				"must be enabled to query the blockchain (specify --txindex in configuration)")
		}
		var tokenPkScript []byte
		var tokenPrivkey ecc.PrivateKey
		if types.IsTokenMintTx(&redeemTx) {
			tokenPkScript, err = api.txManager.GetChain().GetCurTokenOwners(redeemTx.TxOut[0].Amount.Id)
			if err != nil {
				return nil, err
			}
			if tokenPrivkeyStr == nil {
				return nil, fmt.Errorf("Token private key must be provided.")
			}
			tprivkeyByte, err := hex.DecodeString(*tokenPrivkeyStr)
			if err != nil {
				return nil, err
			}
			if len(tprivkeyByte) != 32 {
				return nil, fmt.Errorf("error:%d", len(tprivkeyByte))
			}
			tokenPrivkey, _ = ecc.Secp256k1.PrivKeyFromBytes(tprivkeyByte)
		}
		for i := 0; i < len(redeemTx.TxIn); i++ {
			if i == 0 && len(tokenPkScript) > 0 {
				var tkdb txscript.KeyClosure = func(types.Address) (ecc.PrivateKey, bool, error) {
					return tokenPrivkey, true, nil // compressed is true
				}
				sigScript, err := txscript.SignTxOutput(param, &redeemTx, 0, tokenPkScript, txscript.SigHashAll, tkdb, nil, nil, ecc.ECDSA_Secp256k1)
				if err != nil {
					return nil, err
				}
				redeemTx.TxIn[0].SignScript = sigScript
				continue
			}
			txHash := redeemTx.TxIn[i].PreviousOut.Hash
			// Look up the location of the transaction.
			blockRegion, err := txIndex.TxBlockRegion(txHash)
			if err != nil {
				return nil, errors.New("Failed to retrieve transaction location")
			}
			if blockRegion == nil {
				return nil, rpc.RpcNoTxInfoError(&txHash)
			}

			// Load the raw transaction bytes from the database.
			var txBytes []byte
			err = api.txManager.db.View(func(dbTx database.Tx) error {
				var err error
				txBytes, err = dbTx.FetchBlockRegion(blockRegion)
				return err
			})
			if err != nil {
				return nil, rpc.RpcNoTxInfoError(&txHash)
			}
			// Deserialize the transaction.
			var prevTx types.Transaction
			err = prevTx.Deserialize(bytes.NewReader(txBytes))
			if err != nil {
				return nil, err
			}

			if redeemTx.TxIn[i].PreviousOut.OutIndex >= uint32(len(prevTx.TxOut)) {
				return nil, fmt.Errorf("index:%d", redeemTx.TxIn[i].PreviousOut.OutIndex)
			}

			//
			blockNode := api.txManager.GetChain().BlockDAG().GetBlock(blockRegion.Hash)
			if blockNode == nil {
				return nil, fmt.Errorf("Can't find block %s", blockRegion.Hash)
			}

			if blockNode.GetState().GetStatus().KnownInvalid() {
				return nil, fmt.Errorf("Vin is  illegal %s", blockRegion.Hash)
			}

			pks := prevTx.TxOut[redeemTx.TxIn[i].PreviousOut.OutIndex].PkScript
			sigScript, err := txscript.SignTxOutput(param, &redeemTx, i, pks, txscript.SigHashAll, kdb, nil, nil, ecc.ECDSA_Secp256k1)
			if err != nil {
				return nil, err
			}
			redeemTx.TxIn[i].SignScript = sigScript
		}
	}

	mtxHex, err := marshal.MessageToHex(&redeemTx)
	if err != nil {
		return nil, err
	}
	return mtxHex, nil
}

// token
func (api *PublicTxAPI) CreateTokenRawTransaction(txtype string, coinId uint16, coinName *string, owners *string, uplimit *uint64, inputs []json.TransactionInput, amounts json.Amounts, feeType uint16, feeValue int64) (interface{}, error) {
	txt := types.TxTypeTokenRegulation
	if !strings.HasPrefix(txtype, "0x") {
		switch txtype {
		case "new":
			txt = types.TxTypeTokenNew
		case "renew":
			txt = types.TxTypeTokenRenew
		case "validate":
			txt = types.TxTypeTokenValidate
		case "invalidate":
			txt = types.TxTypeTokenInvalidate
		case "mint":
			txt = types.TxTypeTokenMint
		default:
			return nil, fmt.Errorf("No support %s\n", txtype)
		}
	} else {
		txtype = txtype[2:]
		txtI, err := strconv.ParseInt(txtype, 16, 32)
		if err != nil {
			return nil, err
		}
		txt = types.TxType(txtI)
	}

	mtx := types.NewTransaction()
	mtx.AddTxIn(&types.TxInput{
		PreviousOut: *types.NewOutPoint(&hash.ZeroHash, types.SupperPrevOutIndex),
		Sequence:    uint32(txt),
	})

	if types.CoinID(coinId) <= types.QitmeerReservedID {
		return nil, fmt.Errorf("Coin ID (%d) is qitmeer reserved. It has to be greater than %d for token type update.\n", coinId, types.QitmeerReservedID)
	}

	//
	if txt == types.TxTypeTokenMint {
		if len(inputs) <= 0 {
			return nil, fmt.Errorf("Tx inputs cannot be empty\n")
		}
		if len(amounts) <= 0 {
			return nil, fmt.Errorf("Token amounts cannot be empty\n")
		}

		lockMeer := int64(0)
		for _, input := range inputs {
			txid, err := hash.NewHashFromStr(input.Txid)
			if err != nil {
				return nil, rpc.RpcDecodeHexError(input.Txid)
			}
			prevOut := types.NewOutPoint(txid, input.Vout)
			txIn := types.NewTxInput(prevOut, []byte{})

			entry, err := api.txManager.GetChain().FetchUtxoEntry(*prevOut)
			if err != nil {
				return nil, rpc.RpcNoTxInfoError(txid)
			}
			if entry == nil || entry.IsSpent() {
				return nil, fmt.Errorf("Input(%s %d) is invalid\n", prevOut.Hash, prevOut.OutIndex)
			}
			if !entry.Amount().Id.IsBase() {
				return nil, fmt.Errorf("Token transaction input (%s %d) must be MEERA\n", txIn.PreviousOut.Hash, txIn.PreviousOut.OutIndex)
			}
			lockMeer += entry.Amount().Value
			mtx.AddTxIn(txIn)
		}
		dbnamespace.ByteOrder.PutUint64(mtx.TxIn[0].PreviousOut.Hash[0:8], uint64(lockMeer))

		err := types.CheckCoinID(types.CoinID(coinId))
		if err != nil {
			return nil, err
		}
		for encodedAddr, amount := range amounts {
			// Ensure amount is in the valid range for monetary amounts.
			if amount <= 0 || amount > types.MaxAmount {
				return nil, rpc.RpcInvalidError("Invalid amount: 0 >= %v "+
					"> %v", amount, types.MaxAmount)
			}

			// Decode the provided address.
			addr, err := address.DecodeAddress(encodedAddr)
			if err != nil {
				return nil, rpc.RpcAddressKeyError("Could not decode "+
					"address: %v", err)
			}

			if !address.IsForNetwork(addr, api.txManager.consensus.Params()) {
				return nil, rpc.RpcAddressKeyError("Wrong network: %v",
					addr)
			}

			// Create a new script which pays to the provided address.
			pkScript, err := txscript.PayToAddrScript(addr)
			if err != nil {
				return nil, rpc.RpcInternalError(err.Error(),
					"Pay to address script")
			}

			txOut := types.NewTxOutput(types.Amount{Value: int64(amount), Id: types.CoinID(coinId)}, pkScript)
			mtx.AddTxOut(txOut)
		}
	} else {
		//

		upLi := uint64(math.MaxInt64)
		if uplimit != nil {
			upLi = *uplimit
			if *uplimit == 0 {
				upLi = uint64(math.MaxInt64)
			}
		}

		if coinName != nil {
			if len(*coinName) > token.MaxTokenNameLength {
				return nil, fmt.Errorf("Coin name is too long:%d  (max:%d)", len(*coinName), token.MaxTokenNameLength)
			}
		}
		if txt != types.TxTypeTokenNew {
			err := types.CheckCoinID(types.CoinID(coinId))
			if err != nil {
				return nil, err
			}
		}
		if txt == types.TxTypeTokenNew || txt == types.TxTypeTokenRenew {
			if owners == nil {
				return nil, fmt.Errorf("No owners address\n")
			}
			addr, err := address.DecodeAddress(*owners)
			if err != nil {
				return nil, rpc.RpcAddressKeyError("Could not decode address: %v", err)
			}
			if !address.IsForNetwork(addr, params.ActiveNetParams.Params) {
				return nil, rpc.RpcAddressKeyError("Wrong network: %v", addr)
			}

			if coinName == nil {
				return nil, fmt.Errorf("No coin name\n")
			}
			fcfg := &token.TokenFeeConfig{Type: types.FeeType(feeType), Value: feeValue}
			pkScript, err := txscript.PayToTokenPubKeyHashScript(addr.Script(), types.CoinID(coinId), upLi, *coinName, fcfg.GetData())
			if err != nil {
				return nil, err
			}
			mtx.AddTxOut(&types.TxOutput{PkScript: pkScript})
		} else {
			state := api.txManager.GetChain().GetCurTokenState()
			if state == nil {
				return nil, fmt.Errorf("Token state error\n")
			}
			tt, ok := state.Types[types.CoinID(coinId)]
			if !ok {
				return nil, fmt.Errorf("It doesn't exist: Coin id (%d)\n", coinId)
			}
			if tt.Enable && txt == types.TxTypeTokenValidate {
				return nil, fmt.Errorf("Validate is allowed only when disable: Coin id (%d)\n", coinId)
			}
			if !tt.Enable && txt == types.TxTypeTokenInvalidate {
				return nil, fmt.Errorf("Invalidate is allowed only when enable: Coin id (%d)\n", coinId)
			}
			addr := tt.GetAddress()
			if addr == nil {
				return nil, fmt.Errorf("Token owners is error\n")
			}
			pkScript, err := txscript.PayToTokenPubKeyHashScript(addr.Script(), types.CoinID(coinId), 0, "", 0)
			if err != nil {
				return nil, err
			}
			mtx.AddTxOut(&types.TxOutput{PkScript: pkScript})
		}
	}

	mtxHex, err := marshal.MessageToHex(mtx)
	if err != nil {
		return nil, err
	}
	return mtxHex, nil
}

// cross chain import tx
func (api *PublicTxAPI) CreateImportRawTransaction(pkAddress string, amount int64) (interface{}, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("Amount is empty")
	}
	mtx := types.NewTransaction()
	var buff bytes.Buffer
	err := s.WriteElements(&buff, mtx.Timestamp.Unix())
	if err != nil {
		return nil, err
	}
	th := hash.MustBytesToHash(buff.Bytes())
	mtx.AddTxIn(&types.TxInput{
		PreviousOut: *types.NewOutPoint(&th, types.SupperPrevOutIndex),
		Sequence:    uint32(types.TxTypeCrossChainImport),
	})

	addr, err := address.DecodeAddress(pkAddress)
	if err != nil {
		return nil, rpc.RpcAddressKeyError("Could not decode "+
			"address: %v", err)
	}
	if !address.IsForNetwork(addr, api.txManager.consensus.Params()) {
		return nil, rpc.RpcAddressKeyError("Wrong network: %v",
			addr)
	}
	pkAddr, ok := addr.(*address.SecpPubKeyAddress)
	if !ok {
		return nil, rpc.RpcAddressKeyError("Wrong address: %v", addr)
	}

	pkScript, err := txscript.PayToAddrScript(pkAddr.PKHAddress())
	if err != nil {
		return nil, rpc.RpcInternalError(err.Error(),
			"Pay to address script")
	}

	mtx.AddTxOut(&types.TxOutput{
		Amount:   types.Amount{Id: types.MEERA, Value: amount},
		PkScript: pkScript,
	})

	pkaScript, err := txscript.NewScriptBuilder().AddData([]byte(pkAddress)).Script()
	if err != nil {
		return nil, err
	}
	mtx.TxIn[0].SignScript = pkaScript

	mtxHex, err := marshal.MessageToHex(mtx)
	if err != nil {
		return nil, err
	}
	return mtxHex, nil
}

// cross chain export tx
func (api *PublicTxAPI) CreateExportRawTransaction(txid string, vout uint32, pkAddress string, amount int64) (interface{}, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("Amount is empty")
	}

	aa := json.AdreesAmount{}
	aa[pkAddress] = json.Amout{CoinId: uint16(types.MEERB), Amount: amount}
	inputs := []json.TransactionInput{
		json.TransactionInput{Txid: txid, Vout: vout},
	}
	return api.txManager.CreateRawTransactionV2(inputs, aa, nil)
}

func (api *PublicTxAPI) CreateExportRawTransactionV2(inputs []json.TransactionInput, outputs []json.TransactionOutput, lockTime *int64) (interface{}, error) {
	if len(outputs) <= 0 {
		return nil, fmt.Errorf("outputs number is error")
	}
	ePKAddress := outputs[0].Address
	eAmount := outputs[0].Amount

	if eAmount <= 0 {
		return nil, fmt.Errorf("meerevm amount is empty")
	}
	ePKAddr, err := address.DecodeAddress(ePKAddress)
	if err != nil {
		return nil, rpc.RpcAddressKeyError("Could not decode "+
			"address: %v", err)
	}
	_, ok := ePKAddr.(*address.SecpPubKeyAddress)
	if !ok {
		return nil, fmt.Errorf("%s is not public key address", ePKAddress)
	}

	aa := json.AdreesAmount{}
	aa[ePKAddress] = json.Amout{CoinId: uint16(types.MEERB), Amount: eAmount}
	if len(outputs) > 0 {
		for k, v := range outputs {
			if k == 0 {
				continue
			}
			aa[v.Address] = json.Amout{CoinId: uint16(types.MEERA), Amount: v.Amount}
		}
	}
	return api.txManager.CreateRawTransactionV2(inputs, aa, lockTime)
}
