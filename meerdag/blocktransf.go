package meerdag

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"sort"
)

// Is there a block in DAG?
func (bd *MeerDAG) HasBlock(h *hash.Hash) bool {
	return bd.GetBlockId(h) != MaxId
}

func (bd *MeerDAG) hasBlock(h *hash.Hash) bool {
	return bd.getBlockId(h) != MaxId
}

// Is there a block in DAG?
func (bd *MeerDAG) hasBlockById(id uint) bool {
	return bd.getBlockById(id) != nil
}

// Is there a block in DAG?
func (bd *MeerDAG) HasBlockById(id uint) bool {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.hasBlockById(id)
}

// Is there some block in DAG?
func (bd *MeerDAG) hasBlocks(ids []uint) bool {
	for _, id := range ids {
		if !bd.hasBlockById(id) {
			return false
		}
	}
	return true
}

// Acquire one block by hash
func (bd *MeerDAG) GetBlock(h *hash.Hash) IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlock(h)
}

// Acquire one block by hash
// Be careful, this is inefficient and cannot be called frequently
func (bd *MeerDAG) getBlock(h *hash.Hash) IBlock {
	return bd.getBlockById(bd.getBlockId(h))
}

func (bd *MeerDAG) GetBlockId(h *hash.Hash) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlockId(h)
}

func (bd *MeerDAG) getBlockId(h *hash.Hash) uint {
	if h == nil {
		return MaxId
	}
	if bd.lastSnapshot.block != nil {
		if bd.lastSnapshot.block.GetHash().IsEqual(h) {
			return bd.lastSnapshot.block.GetID()
		}
	}
	bid, err := DBGetBlockIdByHash(bd.db, h)
	if err != nil {
		return MaxId
	}
	return bid
}

// Acquire one block by hash
func (bd *MeerDAG) GetBlockById(id uint) IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlockById(id)
}

// Acquire one block by id
func (bd *MeerDAG) getBlockById(id uint) IBlock {
	if id == MaxId {
		return nil
	}
	block, ok := bd.blocks[id]
	if !ok {
		b, err := bd.loadBlock(id)
		if err != nil {
			log.Warn("get block", "error", err.Error(), "blockID", id)
		}
		return b
	}
	return block
}

func (bd *MeerDAG) HasLoadedBlock(id uint) bool {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.hasLoadedBlock(id)
}

func (bd *MeerDAG) hasLoadedBlock(id uint) bool {
	if id == MaxId {
		return false
	}
	_, ok := bd.blocks[id]
	return ok
}

// Obtain block hash by global order
func (bd *MeerDAG) GetBlockHashByOrder(order uint) *hash.Hash {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	ib := bd.getBlockByOrder(order)
	if ib != nil {
		return ib.GetHash()
	}
	return nil
}

func (bd *MeerDAG) GetBlockByOrder(order uint) IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlockByOrder(order)
}

func (bd *MeerDAG) getBlockByOrder(order uint) IBlock {
	if order >= MaxBlockOrder {
		return nil
	}
	id, ok := bd.commitOrder[order]
	if ok {
		return bd.getBlockById(id)
	}
	id, err := DBGetBlockIdByOrder(bd.db, order)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return bd.getBlockById(id)
}

// Return the last order block
func (bd *MeerDAG) GetLastBlock() IBlock {
	// TODO
	return bd.GetMainChainTip()
}

// This function need a stable sequence,so call it before sorting the DAG.
// If the h is invalid,the function will become a little inefficient.
func (bd *MeerDAG) GetPrevious(id uint) (uint, error) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	if id == 0 {
		return 0, fmt.Errorf("no pre")
	}
	b := bd.getBlockById(id)
	if b == nil {
		return 0, fmt.Errorf("no pre")
	}
	if b.GetOrder() == 0 {
		return 0, fmt.Errorf("no pre")
	}
	// TODO
	ib := bd.getBlockByOrder(b.GetOrder() - 1)
	if ib != nil {
		return ib.GetID(), nil
	}
	return 0, fmt.Errorf("no pre")
}

func (bd *MeerDAG) GetBlockHash(id uint) *hash.Hash {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	ib := bd.getBlockById(id)
	if ib != nil {
		return ib.GetHash()
	}
	return nil
}

func (bd *MeerDAG) GetMainAncestor(block IBlock, height int64) IBlock {
	if height < 0 || height > int64(block.GetHeight()) {
		return nil
	}

	ib := block

	for ib != nil && int64(ib.GetHeight()) != height {
		if !ib.HasParents() {
			ib = nil
			break
		}
		ib = bd.GetBlockById(ib.GetMainParent())
	}
	return ib
}

func (bd *MeerDAG) RelativeMainAncestor(block IBlock, distance int64) IBlock {
	return bd.GetMainAncestor(block, int64(block.GetHeight())-distance)
}

func (bd *MeerDAG) ValidBlock(block IBlock) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	block.GetState().Valid()
	bd.commitBlock.AddPair(block.GetID(), block)
}

func (bd *MeerDAG) InvalidBlock(block IBlock) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	block.GetState().Invalid()
	bd.commitBlock.AddPair(block.GetID(), block)
}

// GetIdSet
func (bd *MeerDAG) GetIdSet(hs []*hash.Hash) *IdSet {
	result := NewIdSet()

	for _, v := range hs {
		if bd.lastSnapshot.block != nil {
			if bd.lastSnapshot.block.GetHash().IsEqual(v) {
				result.Add(bd.lastSnapshot.block.GetID())
				continue
			}
		}
		bid, err := DBGetBlockIdByHash(bd.db, v)
		if err == nil {
			result.Add(uint(bid))
		} else {
			return nil
		}
	}
	return result
}

// Sort block by id
func (bd *MeerDAG) sortBlock(src []*hash.Hash) []*hash.Hash {

	if len(src) <= 1 {
		return src
	}
	srcBlockS := BlockSlice{}
	for i := 0; i < len(src); i++ {
		ib := bd.getBlock(src[i])
		if ib != nil {
			srcBlockS = append(srcBlockS, ib)
		}
	}
	if len(srcBlockS) >= 2 {
		sort.Sort(srcBlockS)
	}
	result := []*hash.Hash{}
	for i := 0; i < len(srcBlockS); i++ {
		result = append(result, srcBlockS[i].GetHash())
	}
	return result
}

// Sort block by id
func (bd *MeerDAG) SortBlock(src []*hash.Hash) []*hash.Hash {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.sortBlock(src)
}

func (bd *MeerDAG) LocateBlocks(gs *GraphState, maxHashes uint) []*hash.Hash {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.locateBlocks(gs, maxHashes)
}

// Locate all eligible block by current graph state.
func (bd *MeerDAG) locateBlocks(gs *GraphState, maxHashes uint) []*hash.Hash {
	if gs.IsExcellent(bd.getGraphState()) {
		return nil
	}
	queue := []IBlock{}
	fs := NewHashSet()
	tips := bd.getValidTips(false)
	queue = append(queue, tips...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if fs.Has(cur.GetHash()) {
			continue
		}
		if gs.GetTips().Has(cur.GetHash()) || cur.GetID() == 0 {
			continue
		}
		needRec := true
		if cur.HasChildren() {
			for _, v := range bd.getChildren(cur).GetMap() {
				ib := v.(IBlock)
				if gs.GetTips().Has(ib.GetHash()) || !fs.Has(ib.GetHash()) && ib.IsOrdered() {
					needRec = false
					break
				}
			}
		}
		if needRec {
			fs.AddPair(cur.GetHash(), cur)
			if cur.HasParents() {
				for _, v := range bd.getParents(cur).GetMap() {
					value := v.(IBlock)
					ib := value
					if fs.Has(ib.GetHash()) {
						continue
					}
					queue = append(queue, ib)

				}
			}
		}
	}

	fsSlice := BlockSlice{}
	for _, v := range fs.GetMap() {
		value := v.(IBlock)
		ib := value
		if gs.GetTips().Has(ib.GetHash()) {
			continue
		}
		if ib.HasChildren() {
			need := true
			for _, v := range bd.getChildren(ib).GetMap() {
				ib := v.(IBlock)
				if gs.GetTips().Has(ib.GetHash()) {
					need = false
					break
				}
			}
			if !need {
				continue
			}
		}
		if !ib.IsOrdered() {
			continue
		}
		fsSlice = append(fsSlice, ib)
	}

	result := []*hash.Hash{}
	if len(fsSlice) >= 2 {
		sort.Sort(fsSlice)
	}
	for i := 0; i < len(fsSlice); i++ {
		if maxHashes > 0 && i >= int(maxHashes) {
			break
		}
		result = append(result, fsSlice[i].GetHash())
	}
	return result
}

// Fuzzy and quick locate all eligible block by current graph state.
// And maxHashes param cannot be empty.
func (bd *MeerDAG) locateBlocksFuzzy(gs *GraphState, maxHashes uint) []*hash.Hash {
	if gs.IsExcellent(bd.getGraphState()) {
		return nil
	}
	curID := MaxId
	for _, k := range gs.tips {
		gst := bd.getBlock(&k)
		if gst == nil || !gst.IsOrdered() {
			continue
		}
		if gst.GetID() < curID {
			curID = gst.GetID()
		}
	}
	fsSlice := BlockSlice{}
	mainTipID := bd.instance.GetMainChainTipId()
	for ; curID <= mainTipID; curID++ {
		ib := bd.getBlockById(curID)
		if ib == nil {
			continue
		}
		if gs.GetTips().Has(ib.GetHash()) {
			continue
		}
		if !ib.IsOrdered() {
			continue
		}
		fsSlice = append(fsSlice, ib)
		if uint(len(fsSlice)) >= maxHashes {
			break
		}
	}

	result := []*hash.Hash{}
	if len(fsSlice) >= 2 {
		sort.Sort(fsSlice)
	}
	for i := 0; i < len(fsSlice); i++ {
		result = append(result, fsSlice[i].GetHash())
	}
	return result
}

// Return the layer of block,it is stable.
// You can imagine that this is the main chain.
func (bd *MeerDAG) GetLayer(id uint) uint {
	return bd.GetBlockById(id).GetLayer()
}
