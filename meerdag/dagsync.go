package meerdag

import (
	"container/list"
	"github.com/Qitmeer/qng/common/hash"
	"sync"
)

// This parameter can be set according to the size of TCP package(1500) to ensure the transmission stability of the network
const MaxMainLocatorNum = 32

// Synchronization mode
type SyncMode byte

const (
	DirectMode SyncMode = 0
	SubDAGMode SyncMode = 1
)

type DAGSync struct {
	bd *MeerDAG

	// The following fields are used to track the graph state being synced to from
	// peers.
	gsMtx sync.Mutex
	gs    *GraphState
}

// CalcSyncBlocks
func (ds *DAGSync) CalcSyncBlocks(gs *GraphState, locator []*hash.Hash, mode SyncMode, maxHashes uint) ([]*hash.Hash, *hash.Hash) {
	ds.bd.stateLock.Lock()
	defer ds.bd.stateLock.Unlock()

	if mode == DirectMode {
		result := []*hash.Hash{}
		if len(locator) == 0 {
			return result, nil
		}
		return ds.bd.sortBlock(locator), nil
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
	return ds.getBlockChainFromMain(point, maxHashes), point.GetHash()
}

// GetMainLocator
func (ds *DAGSync) GetMainLocator(point *hash.Hash) []*hash.Hash {
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
		locator.PushFront(cur.GetHash())
		cur = ds.bd.getBlockById(cur.GetMainParent())
		if cur == nil {
			break
		}
	}
	if cur.GetID() > endBlock.GetID() {
		halfStart := cur.GetID()
		halfEnd := endBlock.GetID()
		hlocator := []*hash.Hash{}
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
			halfEnd = nextID
		}
		if len(hlocator) > 0 {
			for i := len(hlocator) - 1; i >= 0; i-- {
				locator.PushFront(hlocator[i])
			}
		}
	}
	result := []*hash.Hash{endBlock.GetHash()}
	for i := locator.Front(); i != nil; i = i.Next() {
		result = append(result, i.Value.(*hash.Hash))
	}
	return result
}

func (ds *DAGSync) getBlockChainFromMain(point IBlock, maxHashes uint) []*hash.Hash {
	mainTip := ds.bd.getMainChainTip()
	result := []*hash.Hash{}
	for i := point.GetOrder() + 1; i <= mainTip.GetOrder(); i++ {
		block := ds.bd.getBlockByOrder(i)
		if block == nil {
			continue
		}
		result = append(result, block.GetHash())
		if uint(len(result)) >= maxHashes {
			break
		}
	}
	return result
}

func (ds *DAGSync) SetGraphState(gs *GraphState) {
	ds.gsMtx.Lock()
	defer ds.gsMtx.Unlock()

	ds.gs = gs
}

// NewDAGSync
func NewDAGSync(bd *MeerDAG) *DAGSync {
	return &DAGSync{bd: bd}
}
