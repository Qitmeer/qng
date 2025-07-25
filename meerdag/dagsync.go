package meerdag

import (
	"container/list"
	"fmt"
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/ethereum/go-ethereum/core/state"
	"sync"
)

// This parameter can be set according to the size of TCP package(1500) to ensure the transmission stability of the network
const MaxMainLocatorNum = 32

const MaxSnapSyncTargetDepth = 20000
const SnapSyncEVMTargetValve = state.TriesInMemory / 3

// Synchronization mode
type SyncMode byte

const (
	DirectMode   SyncMode = 0
	SubDAGMode   SyncMode = 1
	ContinueMode SyncMode = 2
)

type DAGSync struct {
	bd *MeerDAG

	// The following fields are used to track the graph state being synced to from
	// peers.
	gsMtx sync.Mutex
}

// CalcSyncBlocks
func (ds *DAGSync) CalcSyncBlocks(locator []*hash.Hash, mode SyncMode, maxHashes uint) ([]*hash.Hash, bool, *hash.Hash) {
	ds.bd.stateLock.Lock()
	defer ds.bd.stateLock.Unlock()

	if mode == DirectMode {
		result := []*hash.Hash{}
		if len(locator) == 0 {
			return result, false, nil
		}
		return ds.bd.sortBlock(locator), false, nil
	} else if mode == ContinueMode {
		result := []*hash.Hash{}
		if len(locator) <= 0 {
			return result, false, nil
		}
		point := ds.bd.getBlock(locator[0])
		if point == nil {
			return result, false, nil
		}
		if !ds.bd.isOnMainChain(point.GetID()) {
			return result, false, nil
		}
		startBlock := ds.bd.getBlock(locator[1])
		if startBlock == nil {
			return result, false, nil
		}
		blocks, complete := ds.getBlockChainFromMain(startBlock, maxHashes)
		return blocks, complete, point.GetHash()
	}

	var point IBlock
	for i := len(locator) - 1; i >= 0; i-- {
		mainBlock := ds.bd.getBlock(locator[i])
		if mainBlock == nil {
			continue
		}
		if !ds.bd.isOnMainChain(mainBlock.GetID()) {
			continue
		}
		point = mainBlock
		break
	}

	if point == nil && len(locator) > 0 {
		point = ds.bd.getBlock(locator[0])
		if point != nil {
			for !ds.bd.isOnMainChain(point.GetID()) {
				if point.GetMainParent() == MaxId {
					break
				}
				point = ds.bd.getBlockById(point.GetMainParent())
				if point == nil {
					break
				}
			}
		}

	}

	if point == nil {
		point = ds.bd.getGenesis()
	}
	blocks, complete := ds.getBlockChainFromMain(point, maxHashes)
	return blocks, complete, point.GetHash()
}

// GetMainLocator
func (ds *DAGSync) GetMainLocator(point *hash.Hash, withRoot bool) ([]*hash.Hash, []*hash.Hash) {
	ds.bd.stateLock.Lock()
	defer ds.bd.stateLock.Unlock()

	var endBlock IBlock
	if point != nil {
		endBlock = ds.bd.getBlock(point)
	}
	if endBlock != nil {
		for !ds.bd.isOnMainChain(endBlock.GetID()) {
			if endBlock.GetMainParent() == MaxId {
				break
			}
			endBlock = ds.bd.getBlockById(endBlock.GetMainParent())
			if endBlock == nil {
				break
			}
		}
	}
	if endBlock == nil {
		endBlock = ds.bd.getGenesis()
	}
	startBlock := ds.bd.getMainChainTip()
	locator := list.New()
	cur := startBlock
	const DefaultMainLocatorNum = 10
	for i := 0; i < DefaultMainLocatorNum; i++ {
		if cur.GetID() == 0 ||
			cur.GetMainParent() == MaxId ||
			cur.GetID() <= endBlock.GetID() {
			break
		}
		if withRoot {
			locator.PushFront([]*hash.Hash{cur.GetHash(), cur.GetState().Root()})
		} else {
			locator.PushFront(cur.GetHash())
		}

		cur = ds.bd.getBlockById(cur.GetMainParent())
		if cur == nil {
			break
		}
	}
	if cur.GetID() > endBlock.GetID() {
		halfStart := cur.GetID()
		halfEnd := endBlock.GetID()
		hlocator := []*hash.Hash{}
		var stateRoots []*hash.Hash
		if withRoot {
			stateRoots = []*hash.Hash{}
		}
		for locator.Len()+len(hlocator)+1 < MaxMainLocatorNum {
			//for {
			nextID := (halfStart - halfEnd) / 2
			if nextID <= 0 {
				break
			}
			nextID += halfEnd
			if nextID == halfStart ||
				nextID == halfEnd {
				break
			}
			if !ds.bd.isOnMainChain(nextID) {
				halfEnd++
				continue
			}
			ib := ds.bd.getBlockById(nextID)
			if ib == nil {
				halfEnd++
				continue
			}
			hlocator = append(hlocator, ib.GetHash())
			if withRoot {
				stateRoots = append(stateRoots, ib.GetState().Root())
			}
			halfEnd = nextID
		}
		if len(hlocator) > 0 {
			for i := len(hlocator) - 1; i >= 0; i-- {
				if withRoot {
					locator.PushFront([]*hash.Hash{hlocator[i], stateRoots[i]})
				} else {
					locator.PushFront(hlocator[i])
				}
			}
		}
	}
	result0 := []*hash.Hash{endBlock.GetHash()}
	var result1 []*hash.Hash
	if withRoot {
		result1 = []*hash.Hash{endBlock.GetState().Root()}
	}

	for i := locator.Front(); i != nil; i = i.Next() {
		if withRoot {
			value := i.Value.([]*hash.Hash)
			result0 = append(result0, value[0])
			result1 = append(result1, value[1])
		} else {
			result0 = append(result0, i.Value.(*hash.Hash))
		}
	}
	return result0, result1
}

func (ds *DAGSync) getBlockChainFromMain(point IBlock, maxHashes uint) ([]*hash.Hash, bool) {
	mainTip := ds.bd.getMainChainTip()
	result := []*hash.Hash{}
	complete := true
	for i := point.GetOrder() + 1; i <= mainTip.GetOrder(); i++ {
		block := ds.bd.getBlockByOrder(i)
		if block == nil {
			continue
		}
		result = append(result, block.GetHash())
		if uint(len(result)) >= maxHashes {
			complete = false
			break
		}
	}
	rlen := len(result)
	if uint(rlen) < maxHashes {
		pdb, ok := ds.bd.GetInstance().(*Phantom)
		if ok {
			if !pdb.GetDiffAnticone().IsEmpty() {
				da := pdb.GetDiffAnticone().SortList(false)
				for _, id := range da {
					block := ds.bd.getBlockById(id)
					if block == nil {
						continue
					}
					if ds.bd.CanPrune(block, mainTip) {
						continue
					}
					result = append(result, block.GetHash())
					if uint(len(result)) >= maxHashes {
						complete = false
						break
					}
				}
			}
		}
	}
	return result, complete
}

func (ds *DAGSync) CalcSnapSyncBlocks(locator []*SnapLocator, maxHashes uint, target *SnapLocator, hasMeerState func(*api.HashOrNumber) bool) ([]IBlock, IBlock, error) {
	ds.bd.stateLock.Lock()
	defer ds.bd.stateLock.Unlock()

	mp := ds.bd.getMainChainTip()
	// get target
	if mp.GetOrder() < MaxSnapSyncTargetDepth {
		return nil, nil, fmt.Errorf("No blocks for snap-sync:order=%d", mp.GetOrder())
	}
	if mp.GetState().GetEVMNumber() < state.TriesInMemory {
		return nil, nil, fmt.Errorf("No blocks for snap-sync:number=%d", mp.GetState().GetEVMNumber())
	}
	var targetBlock IBlock
	var oldTargetBlock IBlock
	if target != nil {
		targetBlock = ds.bd.getBlock(target.block)
		if targetBlock == nil {
			return nil, nil, fmt.Errorf("No target:%s", target.block.String())
		}
		if !hasMeerState(api.NewHashOrNumberByNumber(CalculatePivot(targetBlock.GetState().GetEVMNumber()))) {
			oldTargetBlock = targetBlock
			targetBlock = nil
		}
		if targetBlock != nil {
			if mp.GetOrder()-targetBlock.GetOrder() >= MaxSnapSyncTargetDepth {
				oldTargetBlock = targetBlock
				targetBlock = nil
			} else {
				if len(locator) == 1 && locator[0].block.IsEqual(target.block) {
					return nil, targetBlock, nil
				}
			}
		}
	}
	if targetBlock == nil {
		for targetOrder := mp.GetOrder() - StableConfirmations; targetOrder > 0; targetOrder-- {
			if oldTargetBlock != nil {
				if targetOrder < oldTargetBlock.GetOrder() {
					break
				}
			}
			targetID := ds.bd.getBlockIDByOrder(targetOrder)
			if ds.bd.isOnMainChain(targetID) {
				targetB := ds.bd.getBlockById(targetID)
				if targetB == nil {
					continue
				}
				if !hasMeerState(api.NewHashOrNumberByNumber(CalculatePivot(targetB.GetState().GetEVMNumber()))) {
					continue
				}
				targetBlock = targetB
				break
			}
		}
	}

	if targetBlock == nil {
		return nil, nil, fmt.Errorf("No blocks for snap-sync:%d", mp.GetOrder())
	}

	if target != nil {
		if len(locator) != 1 {
			return nil, nil, fmt.Errorf("Locator len error")
		}
		point := ds.bd.getBlock(locator[0].block)
		if point == nil {
			return nil, nil, fmt.Errorf("No block:%s", locator[0].block.String())
		}
		if !point.GetState().Root().IsEqual(locator[0].root) {
			return nil, nil, fmt.Errorf("Target root inconsistent:%s != %s", point.GetState().Root().String(), locator[0].root.String())
		}
		if point.GetOrder() >= targetBlock.GetOrder() {
			return nil, nil, fmt.Errorf("Start point is invalid:%d >= %d(target)", point.GetOrder(), targetBlock.GetOrder())
		}
		return ds.getBlockChainForSnapSync(point, targetBlock, maxHashes), targetBlock, nil
	}

	var point IBlock
	for i := len(locator) - 1; i >= 0; i-- {
		mainBlock := ds.bd.getBlock(locator[i].block)
		if mainBlock == nil {
			continue
		}
		if !mainBlock.GetState().Root().IsEqual(locator[i].root) {
			continue
		}
		point = mainBlock
		break
	}

	if point == nil {
		point = ds.bd.getGenesis()
	}
	if point.GetOrder() >= targetBlock.GetOrder() {
		return nil, nil, fmt.Errorf("Start point is invalid:%d >= %d(target)", point.GetOrder(), targetBlock.GetOrder())
	}
	return ds.getBlockChainForSnapSync(point, targetBlock, maxHashes), targetBlock, nil
}

func (ds *DAGSync) getBlockChainForSnapSync(point IBlock, target IBlock, maxHashes uint) []IBlock {
	result := []IBlock{}
	for i := point.GetOrder() + 1; i <= target.GetOrder(); i++ {
		block := ds.bd.getBlockByOrder(i)
		if block == nil {
			log.Warn("No block", "order", i)
			return result
		}
		result = append(result, block)
		if uint(len(result)) >= maxHashes {
			break
		}
	}
	return result
}

func (ds *DAGSync) TraverseBlocks(start IBlock, fn func(block IBlock) bool) bool {
	pdb, ok := ds.bd.GetInstance().(*Phantom)
	if !ok {
		return true
	}
	mainTip := ds.bd.GetMainChainTip()
	find := false
	for i := start.GetOrder() + 1; i <= mainTip.GetOrder(); i++ {
		find = true
		block := ds.bd.GetBlockByOrder(i)
		if block == nil {
			log.Error("No block by traverse", "order", i)
			return true
		}
		if !fn(block) {
			return false
		}
	}
	if !pdb.GetDiffAnticone().IsEmpty() {
		da := pdb.GetDiffAnticone().SortList(false)
		for i, id := range da {
			if id == start.GetID() {
				find = true
				continue
			}
			if !find {
				continue
			}
			block := ds.bd.GetBlockById(id)
			if block == nil {
				log.Error("No block by traverse", "order", i)
				return true
			}
			if !fn(block) {
				return false
			}
		}
	}
	return true
}

// NewDAGSync
func NewDAGSync(bd *MeerDAG) *DAGSync {
	return &DAGSync{bd: bd}
}

type SnapLocator struct {
	block *hash.Hash
	root  *hash.Hash
}

func NewSnapLocator(block *hash.Hash, root *hash.Hash) *SnapLocator {
	return &SnapLocator{
		block: block,
		root:  root,
	}
}

func CalculatePivot(number uint64) uint64 {
	return number - state.TriesInMemory/2
}
