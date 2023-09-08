/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package meerdag

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/merkle"
	"math"
)

// return the terminal blocks, because there maybe more than one, so this is a set.
func (bd *MeerDAG) GetTips() *HashSet {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	tips := NewHashSet()
	for k := range bd.tips.GetMap() {
		ib := bd.getBlockById(k)
		tips.AddPair(ib.GetHash(), ib)
	}
	return tips
}

// Acquire the tips array of DAG
func (bd *MeerDAG) GetTipsList() []IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	result := bd.instance.GetTipsList()
	if result != nil {
		return result
	}
	result = []IBlock{}
	for k := range bd.tips.GetMap() {
		result = append(result, bd.getBlockById(k))
	}
	return result
}

func (bd *MeerDAG) GetValidTips(expectPriority int) []*hash.Hash {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	tips := bd.getValidTips(true)

	result := []*hash.Hash{tips[0].GetHash()}
	epNum := expectPriority
	for k, v := range tips {
		if k == 0 {
			if bd.GetBlockData(v).GetPriority() <= 1 {
				epNum--
			}
			continue
		}
		if bd.GetBlockData(v).GetPriority() > 1 {
			result = append(result, v.GetHash())
			continue
		}
		if epNum <= 0 {
			break
		}
		result = append(result, v.GetHash())
		epNum--
	}
	return result
}

func (bd *MeerDAG) getValidTips(limit bool) []IBlock {
	temp := bd.tips.Clone()
	mainParent := bd.getMainChainTip()
	temp.Remove(mainParent.GetID())
	var parents []uint
	if temp.Size() > 1 {
		parents = temp.SortHeightList(true)
	} else {
		parents = temp.List()
	}

	tips := []IBlock{mainParent}
	tipsM := map[string]struct{}{}

	for i := 0; i < len(parents); i++ {
		if mainParent.GetID() == parents[i] {
			continue
		}
		block := bd.getBlockById(parents[i])
		if math.Abs(float64(block.GetLayer())-float64(mainParent.GetLayer())) > MaxTipLayerGap {
			continue
		}
		_, exist := tipsM[block.GetHash().String()]
		if exist {
			continue
		}
		tips = append(tips, block)
		tipsM[block.GetHash().String()] = struct{}{}
		if limit && len(tips) >= bd.getMaxParents() {
			break
		}
	}
	return tips
}

// build merkle tree form current DAG tips
func (bd *MeerDAG) BuildMerkleTreeStoreFromTips() []*hash.Hash {
	parents := bd.GetTips().SortList(false)
	return merkle.BuildParentsMerkleTreeStore(parents)
}

// Refresh the dag tip with new block,it will cause changes in tips set.
func (bd *MeerDAG) updateTips(b IBlock) {
	if b.HasChildren() {
		log.Warn(fmt.Sprintf("Cannot add illegal tip:%s", b.GetHash()))
		return
	}
	if bd.tips == nil {
		bd.tips = NewIdSet()
		bd.tips.AddPair(b.GetID(), b)
		return
	}
	for k, v := range bd.tips.GetMap() {
		block := v.(IBlock)
		if block.HasChildren() {
			bd.tips.Remove(k)
		}
	}
	bd.tips.AddPair(b.GetID(), b)
}

func (bd *MeerDAG) optimizeTips() {
	disTipsCount := 0
	for {
		disTips := bd.getDiscardedTips()
		if len(disTips) <= 0 {
			break
		}
		for _, v := range disTips {
			err := bd.removeTip(v)
			if err != nil {
				log.Error(err.Error())
			} else {
				disTipsCount++
				log.Trace(fmt.Sprintf("Remove discarded tip:%d(%s)", v.GetID(), v.GetHash().String()))
			}
		}
	}
	if disTipsCount > 0 {
		log.Trace(fmt.Sprintf("Remove discarded tips:%d", disTipsCount))
	}
}

func (bd *MeerDAG) removeTip(b IBlock) error {
	bd.tips.Remove(b.GetID())
	err := DBDelDAGBlock(bd.db, b.GetID())
	if err != nil {
		return err
	}
	err = DBDelBlockIdByHash(bd.db, b.GetHash())
	if err != nil {
		return err
	}
	err = DBDelDAGTip(bd.db, b.GetID())
	if err != nil {
		return err
	}
	parents := bd.getParents(b)
	for _, v := range parents.GetMap() {
		block := v.(IBlock)
		block.RemoveChild(b.GetID())
		if !block.HasChildren() {
			bd.tips.AddPair(block.GetID(), block)
			err = DBPutDAGTip(bd.db, block.GetID(), block.GetID() == bd.instance.GetMainChainTipId())
			if err != nil {
				return err
			}
		}
		err = DBPutDAGBlock(bd.db, block)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	delete(bd.blocks, b.GetID())

	ph, ok := bd.instance.(*Phantom)
	if !ok {
		return fmt.Errorf("MeerDAG instance error")
	}
	ph.diffAnticone.Remove(b.GetID())
	if ph.virtualBlock.HasParents() {
		ph.virtualBlock.RemoveParent(b.GetID())
	}
	return DBDelDiffAnticone(bd.db, b.GetID())
}

func (bd *MeerDAG) getDiscardedTips() []IBlock {
	mainTip := bd.getMainChainTip()
	var result []IBlock
	for k, v := range bd.tips.GetMap() {
		if k == mainTip.GetID() {
			continue
		}
		block := v.(IBlock)
		if block.IsOrdered() {
			continue
		}
		if block.GetID()+uint(bd.tipsDisLimit) >= mainTip.GetID() {
			continue
		}
		gap := int64(mainTip.GetHeight()) - int64(block.GetHeight())
		if gap > bd.tipsDisLimit {
			if result == nil {
				result = []IBlock{}
			}
			result = append(result, block)
		}
	}
	return result
}

func (bd *MeerDAG) SetTipsDisLimit(limit int64) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	bd.tipsDisLimit = limit
}

func (bd *MeerDAG) GetTipsSet() *IdSet {
	return bd.tips
}
