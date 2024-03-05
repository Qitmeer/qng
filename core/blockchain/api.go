// Copyright (c) 2017-2018 The qitmeer developers

package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/json"
	qjson "github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerdag"
	rapi "github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"strconv"
)

func (b *BlockChain) APIs() []rapi.API {
	apis := b.Service.APIs()
	apis = append(apis, []rapi.API{
		rapi.API{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicBlockAPI(b),
			Public:    true,
		},
	}...)
	return apis
}

type PublicBlockAPI struct {
	chain *BlockChain
}

func NewPublicBlockAPI(bc *BlockChain) *PublicBlockAPI {
	return &PublicBlockAPI{chain: bc}
}

func (api *PublicBlockAPI) GetBlockhash(order int64) (string, error) {
	if order == rapi.LatestBlockOrder.Int64() {
		order = int64(api.chain.BestSnapshot().GraphState.GetMainOrder())
	}
	blockHash, err := api.chain.BlockHashByOrder(uint64(order))
	if err != nil {
		return "", err
	}
	return blockHash.String(), nil
}

// Return the hash range of block from 'start' to 'end'(exclude self)
// if 'end' is equal to zero, 'start' is the number that from the last block to the Gen
// if 'start' is greater than or equal to 'end', it will just return the hash of 'start'
func (api *PublicBlockAPI) GetBlockhashByRange(start int64, end int64) ([]string, error) {
	totalOrder := int64(api.chain.BestSnapshot().GraphState.GetMainOrder())
	if start > totalOrder {
		return nil, fmt.Errorf("startOrder(%d) is greater than or equal to the totalOrder(%d)", start, totalOrder)
	}
	result := []string{}
	if start >= end && end != 0 && end != rapi.LatestBlockOrder.Int64() {
		block, err := api.chain.BlockByOrder(uint64(start))
		if err != nil {
			return nil, err
		}
		result = append(result, block.Hash().String())
	} else if end == 0 {
		for i := totalOrder; i >= 0; i-- {
			if int64(len(result)) >= start {
				break
			}
			block, err := api.chain.BlockByOrder(uint64(i))
			if err != nil {
				return nil, err
			}
			result = append(result, block.Hash().String())
		}
	} else {
		for i := start; i <= totalOrder; i++ {
			if i > end && end != rapi.LatestBlockOrder.Int64() {
				break
			}
			block, err := api.chain.BlockByOrder(uint64(i))
			if err != nil {
				return nil, err
			}
			result = append(result, block.Hash().String())
		}
	}
	return result, nil
}

func (api *PublicBlockAPI) GetBlockByOrder(order int64, verbose *bool, inclTx *bool, fullTx *bool) (interface{}, error) {
	mainOrder := int64(api.chain.BestSnapshot().GraphState.GetMainOrder())
	if order == rapi.LatestBlockOrder.Int64() {
		order = mainOrder
	} else {
		if order > mainOrder {
			return nil, fmt.Errorf("Order is too big")
		}
	}

	blockHash, err := api.chain.BlockHashByOrder(uint64(order))
	if err != nil {
		return nil, err
	}
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	iTx := true
	if inclTx != nil {
		iTx = *inclTx
	}
	fTx := true
	if fullTx != nil {
		fTx = *fullTx
	}
	return api.GetBlock(*blockHash, &vb, &iTx, &fTx)
}

func (api *PublicBlockAPI) GetBlock(h hash.Hash, verbose *bool, inclTx *bool, fullTx *bool) (interface{}, error) {
	return api.makeBlock(h, verbose, inclTx, fullTx, false)
}

func (api *PublicBlockAPI) GetBlockV2(h hash.Hash, verbose *bool, inclTx *bool, fullTx *bool) (interface{}, error) {
	return api.makeBlock(h, verbose, inclTx, fullTx, true)
}

func (api *PublicBlockAPI) GetBestBlockHash() (interface{}, error) {
	best := api.chain.BestSnapshot()
	return best.Hash.String(), nil
}

// The total ordered Block count
func (api *PublicBlockAPI) GetBlockCount() (interface{}, error) {
	best := api.chain.BestSnapshot()
	return best.GraphState.GetMainOrder() + 1, nil
}

// The total Block count, included possible blocks have not ordered by BlockDAG consensus yet at the moments.
func (api *PublicBlockAPI) GetBlockTotal() (interface{}, error) {
	best := api.chain.BestSnapshot()
	return best.GraphState.GetTotal(), nil
}

// GetBlockHeader implements the getblockheader command.
func (api *PublicBlockAPI) GetBlockHeader(hash hash.Hash, verbose bool) (interface{}, error) {

	// Fetch the block node
	node := api.chain.BlockDAG().GetBlock(&hash)
	if node == nil {
		return nil, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: %v", hash))
	}
	// Fetch the header from chain.
	blockHeader, err := api.chain.HeaderByHash(&hash)
	if err != nil {
		return nil, internalError(err.Error(), fmt.Sprintf("Block not found: %v", hash))
	}

	// When the verbose flag isn't set, simply return the serialized block
	// header as a hex-encoded string.
	if !verbose {
		var headerBuf bytes.Buffer
		err := blockHeader.Serialize(&headerBuf)
		if err != nil {
			context := "Failed to serialize block header"
			return nil, internalError(err.Error(), context)
		}
		return hex.EncodeToString(headerBuf.Bytes()), nil
	}
	// Get next block hash unless there are none.
	confirmations := int64(api.chain.BlockDAG().GetConfirmations(node.GetID()))
	layer := api.chain.BlockDAG().GetLayer(node.GetID())
	blockHeaderReply := json.GetBlockHeaderVerboseResult{
		Hash:          hash.String(),
		Confirmations: confirmations,
		Version:       int32(blockHeader.Version),
		ParentRoot:    blockHeader.ParentRoot.String(),
		TxRoot:        blockHeader.TxRoot.String(),
		StateRoot:     blockHeader.StateRoot.String(),
		Difficulty:    blockHeader.Difficulty,
		Layer:         uint32(layer),
		Time:          blockHeader.Timestamp.Unix(),
		PowResult:     blockHeader.Pow.GetPowResult(),
	}

	return blockHeaderReply, nil

}

// Query whether a given block is on the main chain.
// Note that some DAG protocols may not support this feature.
func (api *PublicBlockAPI) IsOnMainChain(h hash.Hash) (interface{}, error) {
	node := api.chain.BlockDAG().GetBlock(&h)
	if node == nil {
		return nil, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: %v", h))
	}
	isOn := api.chain.BlockDAG().IsOnMainChain(node.GetID())

	return strconv.FormatBool(isOn), nil
}

// Return the current height of DAG main chain
func (api *PublicBlockAPI) GetMainChainHeight() (interface{}, error) {
	return strconv.FormatUint(uint64(api.chain.BlockDAG().GetMainChainTip().GetHeight()), 10), nil
}

// Return the weight of block
func (api *PublicBlockAPI) GetBlockWeight(h hash.Hash) (interface{}, error) {
	block, err := api.chain.FetchBlockByHash(&h)
	if err != nil {
		return nil, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: %v", h))
	}
	return strconv.FormatInt(int64(types.GetBlockWeight(block.Block())), 10), nil
}

// Return the total number of orphan blocks, orphan block are the blocks have not been included into the DAG at this moment.
func (api *PublicBlockAPI) GetOrphansTotal() (interface{}, error) {
	return api.chain.GetOrphansTotal(), nil
}

// Obsoleted GetBlockByID Method, since the confused naming, replaced by GetBlockByNum method
func (api *PublicBlockAPI) GetBlockByID(id uint64, verbose *bool, inclTx *bool, fullTx *bool) (interface{}, error) {
	blockHash := api.chain.BlockDAG().GetBlockHash(uint(id))
	if blockHash == nil {
		return nil, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: id=%v", id))
	}
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	iTx := true
	if inclTx != nil {
		iTx = *inclTx
	}
	fTx := true
	if fullTx != nil {
		fTx = *fullTx
	}
	return api.GetBlock(*blockHash, &vb, &iTx, &fTx)
}

// IsBlue:0:not blue;  1：blue  2：Cannot confirm
func (api *PublicBlockAPI) IsBlue(h hash.Hash) (interface{}, error) {
	ib := api.chain.BlockDAG().GetBlock(&h)
	if ib == nil {
		return 2, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: %s", h.String()))
	}
	confirmations := api.chain.BlockDAG().GetConfirmations(ib.GetID())
	if confirmations == 0 {
		return 2, nil
	}
	if api.chain.BlockDAG().IsBlue(ib.GetID()) {
		return 1, nil
	}
	return 0, nil
}

// Return a list hash of the tip blocks of the DAG at this moment.
func (api *PublicBlockAPI) Tips() (interface{}, error) {
	tips := api.chain.BlockDAG().GetTipsList()
	tipinfos := json.TipsInfo{Count: len(tips), Valid: []qjson.TipInfo{}}
	validtips := api.chain.BlockDAG().GetValidTips(meerdag.MaxPriority)
	mtheight := api.chain.BlockDAG().GetMainChainTip().GetHeight()

	temp := map[hash.Hash]struct{}{}
	for _, tiphash := range validtips {
		temp[*tiphash] = struct{}{}
	}
	getTipInfo := func(block meerdag.IBlock, valid bool) qjson.TipInfo {
		pruneHeight := int64(block.GetHeight()) + api.chain.BlockDAG().GetTipsDisLimit()
		pruneCD := pruneHeight - int64(mtheight)
		if pruneCD < 0 {
			pruneCD = 0
		}
		if valid {
			return qjson.TipInfo{ID: uint64(block.GetID()),
				Hash:   block.GetHash().String(),
				Height: uint64(block.GetHeight())}
		} else {
			return qjson.TipInfo{ID: uint64(block.GetID()),
				Hash:        block.GetHash().String(),
				Height:      uint64(block.GetHeight()),
				PruneHeight: uint64(pruneHeight),
				PruneCD:     uint64(pruneCD)}
		}
	}
	for _, tip := range tips {
		if tip.GetHash().IsEqual(validtips[0]) {
			tipinfos.Valid = append(tipinfos.Valid, getTipInfo(tip, true))
			break
		}
	}
	for _, tip := range tips {
		if tip.GetHash().IsEqual(validtips[0]) {
			continue
		}
		if _, ok := temp[*tip.GetHash()]; ok {
			tipinfos.Valid = append(tipinfos.Valid, getTipInfo(tip, true))
		} else {
			if len(tipinfos.Invalid) <= 0 {
				tipinfos.Invalid = []qjson.TipInfo{getTipInfo(tip, false)}
			} else {
				tipinfos.Invalid = append(tipinfos.Invalid, getTipInfo(tip, false))
			}
		}
	}
	return tipinfos, nil
}

// GetCoinbase
func (api *PublicBlockAPI) GetCoinbase(h hash.Hash, verbose *bool) (interface{}, error) {
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	blk, err := api.chain.FetchBlockByHash(&h)
	if err != nil {
		return nil, err
	}
	signDatas, err := txscript.ExtractCoinbaseData(blk.Block().Transactions[0].TxIn[0].SignScript)
	if err != nil {
		return nil, err
	}
	result := []string{}
	for k, v := range signDatas {
		if k < 2 && !vb {
			continue
		}
		result = append(result, hex.EncodeToString(v))
	}
	return result, nil
}

// GetCoinbase
func (api *PublicBlockAPI) GetFees(h hash.Hash) (interface{}, error) {
	feesMap := map[string]int64{}
	fsm := api.chain.GetFees(&h)
	for coinId, v := range fsm {
		if v <= 0 {
			continue
		}
		feesMap[coinId.Name()] = v
	}
	return feesMap, nil
}

func (api *PublicBlockAPI) GetTokenInfo() (interface{}, error) {
	state := api.chain.GetCurTokenState()
	if state == nil {
		return nil, nil
	}

	tbs := []json.TokenState{}
	for _, v := range state.Types {
		ts := json.TokenState{}
		ts.CoinId = uint16(v.Id)
		ts.CoinName = v.Name
		ts.Owners = hex.EncodeToString(v.Owners)
		if v.Id != types.MEERA && v.Id != types.MEERB {
			ts.UpLimit = v.UpLimit
			ts.Enable = v.Enable
			for k, vb := range state.Balances {
				if k == v.Id {
					ts.Balance = vb.Balance
					ts.LockedMeer = vb.LockedMeer
				}
			}
		}
		tbs = append(tbs, ts)
	}
	return tbs, nil
}

func internalError(err, context string) error {
	return fmt.Errorf("%s : %s", context, err)
}

func (api *PublicBlockAPI) GetStateRoot(order int64, verbose *bool) (interface{}, error) {
	mainOrder := int64(api.chain.BestSnapshot().GraphState.GetMainOrder())
	if rapi.BlockOrder(order) == rapi.LatestBlockOrder {
		order = mainOrder
	} else {
		if order > mainOrder {
			return nil, fmt.Errorf("Order is too big")
		} else if order < 0 {
			return nil, fmt.Errorf("Order is too small")
		}
	}
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	ib := api.chain.BlockDAG().GetBlockByOrder(uint(order))
	if ib == nil {
		return nil, internalError(fmt.Errorf("no block").Error(), fmt.Sprintf("Block not found: %d", order))
	}
	api.chain.ChainRLock()
	defer api.chain.ChainRUnlock()
	if vb {
		ret := qjson.OrderedResult{
			qjson.KV{Key: "Hash", Val: ib.GetHash().String()},
			qjson.KV{Key: "Order", Val: order},
			qjson.KV{Key: "Height", Val: ib.GetHeight()},
			qjson.KV{Key: "Valid", Val: !ib.GetState().GetStatus().KnownInvalid()},
			qjson.KV{Key: "EVMStateRoot", Val: ib.GetState().GetEVMRoot().String()},
			qjson.KV{Key: "EVMHeight", Val: ib.GetState().GetEVMNumber()},
			qjson.KV{Key: "EVMHead", Val: ib.GetState().GetEVMHash().String()},
			qjson.KV{Key: "StateRoot", Val: ib.GetState().Root().String()},
		}
		return ret, nil
	}
	return ib.GetState().Root().String(), nil
}

// Get json format block through the EVM block number
// verbose: Show Details or hexadecimal
// inclTx: Show transactions or not
// fullTx: Is complete transaction information displayed
func (api *PublicBlockAPI) GetBlockByNumber(number uint64, verbose *bool, inclTx *bool, fullTx *bool) (interface{}, error) {
	block := api.chain.GetBlockByNumber(number)
	if block == nil {
		return nil, fmt.Errorf("Can't find block: number=%d", number)
	}
	return api.makeBlock(*block.GetHash(), verbose, inclTx, fullTx, true)
}

func (api *PublicBlockAPI) makeBlock(h hash.Hash, verbose *bool, inclTx *bool, fullTx *bool, showFees bool) (interface{}, error) {

	vb := false
	if verbose != nil {
		vb = *verbose
	}
	iTx := true
	if inclTx != nil {
		iTx = *inclTx
	}
	fTx := true
	if fullTx != nil {
		fTx = *fullTx
	}

	block := api.chain.BlockDAG().GetBlock(&h)
	if block == nil {
		return nil, fmt.Errorf("no node")
	}
	node := api.chain.GetBlockNode(block)
	blk := node.block
	// When the verbose flag isn't set, simply return the
	// network-serialized block as a hex-encoded string.
	if !vb {
		blkBytes, err := blk.Bytes()
		if err != nil {
			return nil, internalError(err.Error(),
				"Could not serialize block")
		}
		return hex.EncodeToString(blkBytes), nil
	}
	confirmations := int64(api.chain.BlockDAG().GetConfirmations(block.GetID()))
	bd := api.chain.BlockDAG()
	cs := bd.GetChildren(block)
	children := []*hash.Hash{}
	if cs != nil && !cs.IsEmpty() {
		for _, v := range cs.GetMap() {
			children = append(children, v.(meerdag.IBlock).GetHash())
		}
	}
	api.chain.SetDAGDuplicateTxs(blk, block)
	coinbaseFees := api.chain.CalculateFees(blk)
	coinbaseAmout := types.AmountMap{}

	if showFees {
		coinbaseAmout[blk.Transactions()[0].Tx.TxOut[0].Amount.Id] = blk.Transactions()[0].Tx.TxOut[0].Amount.Value
	} else {
		if coinbaseFees == nil {
			coinbaseAmout[blk.Transactions()[0].Tx.TxOut[0].Amount.Id] = blk.Transactions()[0].Tx.TxOut[0].Amount.Value
		} else {
			coinbaseAmout = coinbaseFees
			coinbaseAmout[blk.Transactions()[0].Tx.TxOut[0].Amount.Id] += blk.Transactions()[0].Tx.TxOut[0].Amount.Value
		}
		coinbaseFees = nil
	}

	//TODO, refactor marshal api
	fields, err := marshal.MarshalJsonBlock(block, blk, iTx, fTx, api.chain.params, confirmations, children, coinbaseAmout, coinbaseFees)
	if err != nil {
		return nil, err
	}
	return fields, nil

}
