package marshal

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	mmeer "github.com/Qitmeer/qng/consensus/model/meer"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/params"
	"strconv"
	"time"
)

// messageToHex serializes a message to the wire protocol encoding using the
// latest protocol version and returns a hex-encoded string of the result.
func MessageToHex(tx *types.Transaction) (string, error) {
	var buf bytes.Buffer
	err := tx.Encode(&buf, protocol.ProtocolVersion, types.TxSerializeFull)
	if err != nil {
		context := fmt.Sprintf("Failed to encode msg of type %T", tx)
		return "", fmt.Errorf("%s : %s", context, err.Error())
	}

	return hex.EncodeToString(buf.Bytes()), nil
}

func MarshalJsonTx(tx *types.Tx, params *params.Params, blkHashStr string,
	confirmations int64, coinbaseAmout types.AmountMap, state bool) (json.TxRawResult, error) {
	if tx == nil {
		return json.TxRawResult{}, errors.New("can't marshal nil transaction")
	}
	return MarshalJsonTransaction(tx, params, blkHashStr, confirmations, coinbaseAmout, state)
}

func MarshalJsonTransaction(transaction *types.Tx, params *params.Params, blkHashStr string,
	confirmations int64, coinbaseAmout types.AmountMap, state bool) (json.TxRawResult, error) {
	tx := transaction.Tx
	hexStr, err := MessageToHex(tx)
	if err != nil {
		return json.TxRawResult{}, err
	}
	txr := json.TxRawResult{
		Hex:       hexStr,
		Txid:      tx.TxHash().String(),
		TxHash:    tx.TxHashFull().String(),
		Size:      int32(tx.SerializeSize()),
		Version:   tx.Version,
		LockTime:  tx.LockTime,
		Expire:    tx.Expire,
		Vin:       MarshJsonVin(tx),
		Duplicate: transaction.IsDuplicate,
		Txsvalid:  state,
		Type:      types.DetermineTxType(tx).String(),
	}
	if tx.Timestamp.Unix() > 0 {
		txr.Timestamp = tx.Timestamp.Format(time.RFC3339)
	}
	if tx.IsCoinBase() {
		txr.Vout = MarshJsonCoinbaseVout(tx, nil, params, coinbaseAmout)
	} else {
		txr.Vout = MarshJsonVout(tx, nil, params)
	}
	if blkHashStr != "" {
		txr.BlockHash = blkHashStr
		txr.Confirmations = confirmations
	}
	return txr, nil
}

func MarshJsonVin(tx *types.Transaction) []json.Vin {
	// Coinbase transactions only have a single txin by definition.
	vinList := make([]json.Vin, len(tx.TxIn))
	if tx.IsCoinBase() {
		txIn := tx.TxIn[0]
		vinEntry := &vinList[0]
		vinEntry.Coinbase = hex.EncodeToString(txIn.SignScript)
		vinEntry.Sequence = txIn.Sequence
		return vinList
	} else if types.IsTokenTx(tx) {
		for i, txIn := range tx.TxIn {
			disbuf, _ := txscript.DisasmString(txIn.SignScript)

			vinEntry := &vinList[i]
			if i == 0 {
				vinEntry.TxType = types.DetermineTxType(tx).String()
			} else {
				vinEntry.Txid = txIn.PreviousOut.Hash.String()
				vinEntry.Vout = txIn.PreviousOut.OutIndex
				vinEntry.Sequence = txIn.Sequence
			}

			vinEntry.ScriptSig = &json.ScriptSig{
				Asm: disbuf,
				Hex: hex.EncodeToString(txIn.SignScript),
			}

		}
		return vinList
	} else if types.IsCrossChainImportTx(tx) {
		ctx, err := mmeer.NewImportTx(tx)
		if err != nil {
			return vinList
		}
		from, err := common.NewMeerEVMAddress(ctx.From)
		if err != nil {
			return vinList
		}
		vinEntry := &vinList[0]
		vinEntry.From = from.String()
		vinEntry.Value = ctx.Value
		vinEntry.TxType = types.DetermineTxType(tx).String()
		return vinList
	} else if types.IsCrossChainVMTx(tx) {
		vinList[0].TxType = types.DetermineTxType(tx).String()
		sig := tx.TxIn[0].SignScript
		// disbuf, _ := txscript.DisasmString(sig)  //TODO, the Disasm is not fully work for the cross-chain tx
		vinList[0].ScriptSig = &json.ScriptSig{
			// Asm: disbuf,
			Hex: common.ToTxHexStr(sig),
		}
		vinList[0].Txid = "0x" + tx.TxIn[0].PreviousOut.Hash.String()
		return vinList
	}

	for i, txIn := range tx.TxIn {
		// The disassembled string will contain [error] inline
		// if the script doesn't fully parse, so ignore the
		// error here.
		disbuf, _ := txscript.DisasmString(txIn.SignScript)

		vinEntry := &vinList[i]
		vinEntry.Txid = txIn.PreviousOut.Hash.String()
		vinEntry.Vout = txIn.PreviousOut.OutIndex
		vinEntry.Sequence = txIn.Sequence
		vinEntry.ScriptSig = &json.ScriptSig{
			Asm: disbuf,
			Hex: hex.EncodeToString(txIn.SignScript),
		}
	}
	return vinList
}

func MarshJsonVout(tx *types.Transaction, filterAddrMap map[string]struct{}, params *params.Params) []json.Vout {
	voutList := make([]json.Vout, 0, len(tx.TxOut))
	for _, v := range tx.TxOut {
		// The disassembled string will contain [error] inline if the
		// script doesn't fully parse, so ignore the error here.
		disbuf, _ := txscript.DisasmString(v.PkScript)

		// Ignore the error here since an error means the script
		// couldn't parse and there is no additional information
		// about it anyways.
		sc, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(
			v.PkScript, params)
		scriptClass := sc.String()

		// Encode the addresses while checking if the address passes the
		// filter when needed.
		passesFilter := len(filterAddrMap) == 0
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

		if !passesFilter {
			continue
		}

		var vout json.Vout
		vout.Coin = v.Amount.Id.Name()
		vout.CoinId = uint16(v.Amount.Id)
		vout.Amount = uint64(v.Amount.Value)
		voutSPK := &vout.ScriptPubKey
		voutSPK.Addresses = encodedAddrs
		if len(encodedAddrs) > 0 {
			if types.CoinID(vout.CoinId) == types.MEERB {
				ctx, err := mmeer.NewExportTx(tx)
				if err == nil {
					to, err := common.NewMeerEVMAddress(ctx.To)
					if err == nil {
						vout.To = to.String()
						voutList = append(voutList, vout)
						continue
					}
				}
			}
		}
		voutSPK.Asm = disbuf
		voutSPK.Type = scriptClass
		voutSPK.ReqSigs = int32(reqSigs)
		voutSPK.Hex = hex.EncodeToString(v.PkScript)
		voutList = append(voutList, vout)
	}

	return voutList
}

func MarshJsonCoinbaseVout(tx *types.Transaction, filterAddrMap map[string]struct{}, params *params.Params, coinbaseAmout types.AmountMap) []json.Vout {
	if len(coinbaseAmout) <= 0 ||
		tx.CachedTxHash().IsEqual(params.GenesisBlock.Transactions[0].CachedTxHash()) {
		return MarshJsonVout(tx, filterAddrMap, params)
	}
	voutList := make([]json.Vout, 0, len(tx.TxOut))
	for k, v := range tx.TxOut {
		isopr := opreturn.IsOPReturn(v.PkScript)
		if k > 0 && coinbaseAmout[v.Amount.Id] <= 0 && !isopr {
			continue
		}
		// The disassembled string will contain [error] inline if the
		// script doesn't fully parse, so ignore the error here.
		disbuf, _ := txscript.DisasmString(v.PkScript)

		// Ignore the error here since an error means the script
		// couldn't parse and there is no additional information
		// about it anyways.
		sc, addrs, reqSigs, _ := txscript.ExtractPkScriptAddrs(
			v.PkScript, params)
		scriptClass := sc.String()

		// Encode the addresses while checking if the address passes the
		// filter when needed.
		passesFilter := len(filterAddrMap) == 0
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

		if !passesFilter {
			continue
		}

		var vout json.Vout
		voutSPK := &vout.ScriptPubKey
		if !isopr {
			vout.Coin = v.Amount.Id.Name()
			vout.CoinId = uint16(v.Amount.Id)
			vout.Amount = uint64(coinbaseAmout[v.Amount.Id])
			voutSPK.Type = scriptClass
		} else {
			opr, err := opreturn.NewOPReturnFrom(v.PkScript)
			if err != nil {
				continue
			}
			voutSPK.Type = opr.GetType().Name()
		}

		voutSPK.Addresses = encodedAddrs
		voutSPK.Asm = disbuf
		voutSPK.Hex = hex.EncodeToString(v.PkScript)

		voutSPK.ReqSigs = int32(reqSigs)
		voutList = append(voutList, vout)
	}

	return voutList
}

// RPCMarshalBlock converts the given block to the RPC output which depends on fullTx. If inclTx is true transactions are
// returned. When fullTx is true the returned block contains full transaction details, otherwise it will only contain
// transaction hashes.
func MarshalJsonBlock(block meerdag.IBlock, b *types.SerializedBlock, inclTx bool, fullTx bool,
	params *params.Params, confirmations int64, children []*hash.Hash, coinbaseAmout types.AmountMap, coinbaseFee types.AmountMap) (json.OrderedResult, error) {
	state := !block.GetState().GetStatus().KnownInvalid()
	isOrdered := block.IsOrdered()

	head := b.Block().Header // copies the header once
	// Get next block hash unless there are none.
	fields := json.OrderedResult{
		{Key: "hash", Val: b.Hash().String()},
		{Key: "txsvalid", Val: state},
		{Key: "confirmations", Val: confirmations},
		{Key: "version", Val: head.Version},
		{Key: "weight", Val: types.GetBlockWeight(b.Block())},
		{Key: "height", Val: b.Height()},
		{Key: "txRoot", Val: head.TxRoot.String()},
	}
	if isOrdered {
		fields = append(fields, json.KV{Key: "order", Val: block.GetOrder()})
	}
	if inclTx {
		formatTx := func(tx *types.Tx) (interface{}, error) {
			return tx.Hash().String(), nil
		}
		if fullTx {
			formatTx = func(tx *types.Tx) (interface{}, error) {
				return MarshalJsonTx(tx, params, b.Hash().String(), confirmations, coinbaseAmout, state)
			}
		}
		txs := b.Transactions()
		transactions := make([]interface{}, len(txs))
		var err error
		for i, tx := range txs {
			if transactions[i], err = formatTx(tx); err != nil {
				return nil, err
			}
		}
		fields = append(fields, json.KV{Key: "transactions", Val: transactions})
	}
	if coinbaseFee != nil {
		fees := []json.Amout{}
		for coinid, amount := range coinbaseFee {
			if amount <= 0 {
				continue
			}
			fees = append(fees, json.Amout{CoinId: uint16(coinid), Amount: amount})
		}
		if len(fees) > 0 {
			fields = append(fields, json.KV{Key: "transactionfee", Val: fees})
		}
	}
	fields = append(fields, json.OrderedResult{
		{Key: "stateRoot", Val: head.StateRoot.String()},
		{Key: "bits", Val: strconv.FormatUint(uint64(head.Difficulty), 16)},
		{Key: "difficulty", Val: head.Difficulty},
		{Key: "pow", Val: head.Pow.GetPowResult()},
		{Key: "timestamp", Val: head.Timestamp.Format(time.RFC3339)},
		{Key: "parentroot", Val: head.ParentRoot.String()},
	}...)
	tempArr := []string{}
	if b.Block().Parents != nil && len(b.Block().Parents) > 0 {

		for i := 0; i < len(b.Block().Parents); i++ {
			tempArr = append(tempArr, b.Block().Parents[i].String())
		}
	} else {
		tempArr = append(tempArr, "null")
	}
	fields = append(fields, json.KV{Key: "parents", Val: tempArr})

	tempArr = []string{}
	if len(children) > 0 {

		for i := 0; i < len(children); i++ {
			tempArr = append(tempArr, children[i].String())
		}
	} else {
		tempArr = append(tempArr, "null")
	}
	fields = append(fields, json.KV{Key: "children", Val: tempArr})

	return fields, nil
}

func GetGraphStateResult(gs *meerdag.GraphState) *json.GetGraphStateResult {
	if gs != nil {
		tips := []string{}
		for k, v := range gs.GetTipsList() {
			if k == 0 {
				tips = append(tips, v.String()+" main")
			} else {
				tips = append(tips, v.String())
			}
		}
		return &json.GetGraphStateResult{
			Tips:       tips,
			MainOrder:  uint32(gs.GetMainOrder()),
			Layer:      uint32(gs.GetLayer()),
			MainHeight: uint32(gs.GetMainHeight()),
		}
	}
	return nil
}
