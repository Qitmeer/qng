package meerdag

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/params"
	"io"
	"math"
	"time"
)

// Load from database
func (bd *MeerDAG) Load(blockTotal uint, genesis *hash.Hash) error {
	err := bd.db.View(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		serializedData := meta.Get(DagInfoBucketName)
		if serializedData == nil {
			return fmt.Errorf("dag load error")
		}
		return bd.Decode(bytes.NewReader(serializedData))
	})
	if err != nil {
		return err
	}
	bd.genesis = *genesis
	bd.blockTotal = blockTotal
	bd.blocks = map[uint]IBlock{}
	bd.tips = NewIdSet()
	return bd.instance.Load()
}

func (bd *MeerDAG) Encode(w io.Writer) error {
	dagTypeIndex := GetDAGTypeIndex(bd.instance.GetName())
	err := s.WriteElements(w, dagTypeIndex)
	if err != nil {
		return err
	}
	return bd.instance.Encode(w)
}

// decode
func (bd *MeerDAG) Decode(r io.Reader) error {
	var dagTypeIndex byte
	err := s.ReadElements(r, &dagTypeIndex)
	if err != nil {
		return err
	}
	if GetDAGTypeIndex(bd.instance.GetName()) != dagTypeIndex {
		return fmt.Errorf("The dag type is %s, but read is %s", bd.instance.GetName(), GetDAGTypeByIndex(dagTypeIndex))
	}
	return bd.instance.Decode(r)
}

func (bd *MeerDAG) GetBlockData(ib IBlock) IBlockData {
	bd.blockDataLock.Lock()
	defer bd.blockDataLock.Unlock()
	if ib.GetID() < bd.blockTotal {
		bd.blockDataCache[ib.GetID()] = time.Now()
	}
	if ib.IsLoaded() {
		return ib.GetData()
	}
	// load
	data := bd.getBlockData(ib.GetHash())
	if data == nil {
		panic(fmt.Errorf("Can't load block data:%s", ib.GetHash().String()))
	}
	ib.SetData(data)
	return ib.GetData()
}

func (bd *MeerDAG) updateBlockDataCache() {
	cacheSize := bd.GetBlockDataCacheSize()
	if cacheSize <= MinBlockDataCache {
		return
	}
	maxLife := params.ActiveNetParams.TargetTimePerBlock
	startTime := time.Now()
	mainHeight := bd.GetMainChainTip().GetHeight()
	need := cacheSize - MinBlockDataCache
	blockDataCache := map[uint]time.Time{}
	bd.blockDataLock.Lock()
	const waitTime = time.Second * 2
	for k, t := range bd.blockDataCache {
		blockDataCache[k] = t
		need--
		if need <= 0 {
			break
		}
		if time.Since(startTime) > waitTime {
			break
		}
	}
	bd.blockDataLock.Unlock()
	for k, t := range blockDataCache {
		if time.Since(t) > maxLife {
			if !bd.HasLoadedBlock(k) {
				bd.blockDataLock.Lock()
				delete(bd.blockDataCache, k)
				bd.blockDataLock.Unlock()
				continue
			}
			ib := bd.GetBlockById(k)
			if ib != nil {
				if math.Abs(float64(mainHeight)-float64(ib.GetHeight())) <= float64(MinBlockDataCache) {
					continue
				}
				ib.SetData(nil)
			}
			bd.blockDataLock.Lock()
			delete(bd.blockDataCache, k)
			bd.blockDataLock.Unlock()
		}
		if time.Since(startTime) > waitTime {
			break
		}
	}
}

func (bd *MeerDAG) LoadBlockDataSet(sets *IdSet) {
	if sets == nil {
		return
	}
	for _, v := range sets.GetMap() {
		bd.GetBlockData(v.(IBlock))
	}
}

func (bd *MeerDAG) GetBlockDataCacheSize() int {
	bd.blockDataLock.Lock()
	defer bd.blockDataLock.Unlock()
	return len(bd.blockDataCache)
}

func (bd *MeerDAG) loadBlock(id uint) (IBlock, error) {
	ph, ok := bd.instance.(*Phantom)
	if !ok {
		return nil, fmt.Errorf("MeerDAG instance error")
	}
	block := Block{id: id}
	ib := ph.CreateBlock(&block)
	err := bd.db.View(func(dbTx database.Tx) error {
		return DBGetDAGBlock(dbTx, ib)
	})
	if err != nil {
		return nil, err
	}
	if id == 0 && !ib.GetHash().IsEqual(ph.bd.GetGenesisHash()) {
		return nil, fmt.Errorf("genesis data mismatch")
	}
	if ib.HasParents() {
		parentIDs := ib.GetParents().List()
		for _, pid := range parentIDs {
			parent, ok := bd.blocks[pid]
			if ok {
				parent.AttachChild(ib)
				ib.AttachParent(parent)
			}
		}
	}
	if ib.HasChildren() {
		childrenIDs := ib.GetChildren().List()
		for _, pid := range childrenIDs {
			child, ok := bd.blocks[pid]
			if ok {
				child.AttachParent(ib)
				ib.AttachChild(child)
			}
		}
	}
	bd.blocks[ib.GetID()] = ib
	return ib, nil
}

func (bd *MeerDAG) GetParents(ib IBlock) *IdSet {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.getParents(ib)
}

// get parents from block if it is not loaded, it will be loaded
func (bd *MeerDAG) getParents(ib IBlock) *IdSet {
	if !ib.HasParents() {
		return ib.GetParents()
	}
	parents := ib.GetParents()
	parentIDs := parents.List()
	for _, pid := range parentIDs {
		if parents.IsDataEmpty(pid) {
			ib.AttachParent(bd.getBlockById(pid))
		}
	}
	return parents
}

func (bd *MeerDAG) GetChildren(ib IBlock) *IdSet {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.getChildren(ib)
}

// get children from block if it is not loaded, it will be loaded
func (bd *MeerDAG) getChildren(ib IBlock) *IdSet {
	if !ib.HasChildren() {
		return ib.GetChildren()
	}
	children := ib.GetChildren()
	childIDs := children.List()
	for _, cid := range childIDs {
		if children.IsDataEmpty(cid) {
			ib.AttachChild(bd.getBlockById(cid))
		}
	}
	return children
}

func (bd *MeerDAG) GetBlockCacheSize() int {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return len(bd.blocks)
}

func (bd *MeerDAG) updateBlockCache() {
	cacheSize := bd.GetBlockCacheSize()
	if cacheSize <= MinBlockPruneSize {
		return
	}
	mainTip := bd.GetMainChainTip()
	need := cacheSize - MinBlockPruneSize
	deletes := []IBlock{}

	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	for k, v := range bd.blocks {
		if k == 0 ||
			k == mainTip.GetID() ||
			bd.tips.Has(k) ||
			v.GetHash().IsEqual(bd.GetGenesisHash()) ||
			bd.commitBlock.Has(k) {
			continue
		}
		if v.GetHeight()+MinBlockPruneSize < mainTip.GetHeight() {
			deletes = append(deletes, v)
		}
		//
		need--
		if need <= 0 {
			break
		}
	}
	for _, b := range deletes {
		bd.unloadBlock(b)
	}
}

func (bd *MeerDAG) unloadBlock(ib IBlock) {
	delete(bd.blocks, ib.GetID())
	if ib.HasParents() {
		for _, parent := range ib.GetParents().GetMap() {
			if parent == nil {
				continue
			}
			pib, ok := parent.(IBlock)
			if !ok {
				continue
			}
			pib.DetachChild(ib)
		}
	}

	if ib.HasChildren() {
		for _, child := range ib.GetChildren().GetMap() {
			if child == nil {
				continue
			}
			cib, ok := child.(IBlock)
			if !ok {
				continue
			}
			cib.DetachParent(ib)
		}
	}

}
