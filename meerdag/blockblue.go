package meerdag

import (
	"github.com/Qitmeer/qng/common/hash"
	"time"
)

// GetBlues
func (bd *MeerDAG) GetBlues(parents *IdSet) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.instance.GetBlues(parents)
}

func (bd *MeerDAG) GetBluesByHash(h *hash.Hash) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBluesByBlock(bd.getBlockById(bd.getBlockId(h)))
}

func (bd *MeerDAG) GetBluesByBlock(ib IBlock) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBluesByBlock(ib)
}

func (bd *MeerDAG) getBluesByBlock(ib IBlock) uint {
	if ib == nil {
		return 0
	}
	pb, ok := ib.(*PhantomBlock)
	if !ok {
		return 0
	}
	return pb.blueNum
}

// IsBlue
func (bd *MeerDAG) IsBlue(id uint) bool {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.instance.IsBlue(id)
}

func (bd *MeerDAG) GetBlueInfoByHash(h *hash.Hash) *BlueInfo {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlueInfo(bd.getBlockById(bd.getBlockId(h)))
}

func (bd *MeerDAG) GetBlueInfo(ib IBlock) *BlueInfo {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getBlueInfo(ib)
}

func (bd *MeerDAG) getBlueInfo(ib IBlock) *BlueInfo {
	if ib == nil {
		return nil
	}
	if ib.GetID() == 0 {
		return NewBlueInfo(0, 0, 0, int64(ib.GetHeight()))
	}
	if !ib.HasParents() {
		return NewBlueInfo(0, 0, 0, int64(ib.GetHeight()))
	}
	if ib.GetMainParent() == 0 {
		return NewBlueInfo(1, 0, 0, int64(ib.GetHeight()))
	}
	mainIB, ok := bd.getParents(ib).Get(ib.GetMainParent()).(IBlock)
	if !ok {
		return NewBlueInfo(1, 0, 0, int64(ib.GetHeight()))
	}
	mt := bd.GetBlockData(ib).GetTimestamp() - bd.GetBlockData(mainIB).GetTimestamp()
	if mt <= 0 {
		mt = 1
	}
	mt *= int64(time.Second)

	pb, ok := ib.(*PhantomBlock)
	if !ok {
		return NewBlueInfo(1, 0, 0, int64(ib.GetHeight()))
	}
	blues := 1
	blues += pb.GetBlueDiffAnticoneSize()
	return NewBlueInfo(pb.blueNum+1, mt/int64(blues), int64(mainIB.GetState().GetWeight()), int64(ib.GetHeight()))
}
