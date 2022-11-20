package meerdag

import (
	"fmt"
	"sync"
)

const (
	VMK_KEY = 1
	RET_KEY = 2
)

// Batch check the blue and mature properties of blocks in views perspective.
// targets: Need check blocks
// views: Block DAG perspective when calculate the result
// max: Max maturity
func (bd *MeerDAG) CheckBlueAndMature(targets []uint, views []uint, max uint) error {
	return bd.doCheckBlueAndMature(targets, views, max, false)
}

// Batch check the blue and mature properties of blocks in views perspective, and enable multithreading mode.
// targets: Need check blocks
// views: Block DAG perspective when calculate the result
// max: Max maturity
func (bd *MeerDAG) CheckBlueAndMatureMT(targets []uint, views []uint, max uint) error {
	return bd.doCheckBlueAndMature(targets, views, max, false)
}

func (bd *MeerDAG) doCheckBlueAndMature(targets []uint, views []uint, max uint, multithreading bool) error {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	targetIBs := []IBlock{}
	maxTargetLayer := uint(0)
	for _, target := range targets {
		if target == MaxId {
			return fmt.Errorf("Target Block ID(%d) is invalid", target)
		}
		targetBlock := bd.getBlockById(target)
		if targetBlock == nil {
			return fmt.Errorf("Target Block ID(%d) is invalid", target)
		}
		targetIBs = append(targetIBs, targetBlock)

		if targetBlock.GetLayer() > maxTargetLayer {
			maxTargetLayer = targetBlock.GetLayer()
		}
	}

	var mainViewIB IBlock
	var maxViewIB IBlock
	var iviews []IBlock
	for _, v := range views {
		ib := bd.getBlockById(v)
		if ib == nil {
			return fmt.Errorf("View Block ID(%d) is invalid", v)
		}
		if maxTargetLayer >= ib.GetLayer() {
			return fmt.Errorf("View Block Hash(%s) is invalid", ib.GetHash().String())
		}

		if maxViewIB == nil || maxViewIB.GetLayer() < ib.GetLayer() {
			maxViewIB = ib
		}

		if mainViewIB == nil && bd.instance.isOnMainChain(ib.GetID()) {
			mainViewIB = ib
		}

		iviews = append(iviews, ib)
	}

	if multithreading {

		resultPro := sync.Map{}
		resultPro.Store(VMK_KEY, nil)
		resultPro.Store(RET_KEY, nil)
		wg := sync.WaitGroup{}
		for _, target := range targetIBs {
			wg.Add(1)
			go func(t IBlock) {
				v, ok := resultPro.Load(VMK_KEY)
				if !ok {
					wg.Done()
					return
				}
				r, ok := resultPro.Load(RET_KEY)
				if !ok {
					wg.Done()
					return
				}
				if r != nil {
					wg.Done()
					return
				}
				var viewMainFork IBlock
				var targetMainFork IBlock
				result := true
				if v != nil {
					viewMainFork = v.(IBlock)
				}
				result, viewMainFork, targetMainFork = bd.processMaturity(t, iviews, mainViewIB, maxViewIB, viewMainFork, max)
				if !result {
					resultPro.Store(RET_KEY, fmt.Errorf("Target Block Hash(%s) is immature", t.GetHash().String()))
				}

				if !bd.instance.(*Phantom).doIsBlue(t, targetMainFork) {
					resultPro.Store(RET_KEY, fmt.Errorf("Target Block Hash(%s) is not blue", t.GetHash().String()))
				}
				if v == nil && viewMainFork != nil {
					resultPro.Store(VMK_KEY, viewMainFork)
				}
				wg.Done()
			}(target)
		}
		wg.Wait()
		r, ok := resultPro.Load(RET_KEY)
		if !ok {
			return fmt.Errorf("unknown error")
		}
		if r != nil {
			return r.(error)
		}
		return nil
	} else {
		var targetMainFork IBlock
		var viewMainFork IBlock
		result := true
		for _, target := range targetIBs {

			result, viewMainFork, targetMainFork = bd.processMaturity(target, iviews, mainViewIB, maxViewIB, viewMainFork, max)
			if !result {
				return fmt.Errorf("Target Block Hash(%s) is immature", target.GetHash().String())
			}
			if !bd.instance.(*Phantom).doIsBlue(target, targetMainFork) {
				return fmt.Errorf("Target Block Hash(%s) is not blue", target.GetHash().String())
			}
		}
		return nil
	}
}

// processMaturity
func (bd *MeerDAG) processMaturity(target IBlock, views []IBlock, mainViewIB IBlock, maxViewIB IBlock, viewMainFork IBlock, max uint) (bool, IBlock, IBlock) {
	//
	if int64(maxViewIB.GetLayer())-int64(target.GetLayer()) < int64(max) {
		return false, nil, nil
	}

	var targetMainFork IBlock
	if bd.instance.isOnMainChain(target.GetID()) {
		targetMainFork = target
	} else {
		targetMainFork = bd.getMainFork(target, true)
	}
	if targetMainFork == nil {
		return false, nil, nil
	}
	if mainViewIB != nil {
		if int64(mainViewIB.GetLayer())-int64(targetMainFork.GetLayer()) >= int64(max) {
			return true, nil, targetMainFork
		}
	}

	if viewMainFork == nil {
		viewMainFork = bd.getMainFork(maxViewIB, false)
	}

	if viewMainFork != nil {
		if int64(viewMainFork.GetLayer())-int64(targetMainFork.GetLayer()) >= int64(max) {
			return true, viewMainFork, targetMainFork
		}
	}
	//
	queueSet := NewIdSet()
	queue := []IBlock{}

	for _, v := range views {
		queue = append(queue, v)
		queueSet.Add(v.GetID())
		//
		if v.GetID() == maxViewIB.GetID() {
			continue
		}
		viewMainFork = bd.getMainFork(v, false)
		if viewMainFork != nil {
			if int64(viewMainFork.GetLayer())-int64(targetMainFork.GetLayer()) >= int64(max) {
				return true, viewMainFork, targetMainFork
			}
		}
	}
	connected := false
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.GetID() == target.GetID() {
			connected = true
			break
		}
		if !cur.HasParents() {
			continue
		}
		if cur.GetLayer() <= target.GetLayer() {
			continue
		}

		for _, v := range bd.getParents(cur).GetMap() {
			ib := v.(IBlock)
			if queueSet.Has(ib.GetID()) {
				continue
			}
			queue = append(queue, ib)
			queueSet.Add(ib.GetID())
		}
	}
	return connected, viewMainFork, targetMainFork
}

// processMaturity
func (bd *MeerDAG) CheckMainBlueAndMature(target IBlock, targetMainFork IBlock, max uint) (bool, IBlock) {
	//
	mainTip := bd.GetMainChainTip()
	if mainTip == nil {
		return false, targetMainFork
	}
	if int64(mainTip.GetLayer())-int64(target.GetLayer()) < int64(max) {
		return false, targetMainFork
	}

	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	if targetMainFork == nil {
		if bd.instance.isOnMainChain(target.GetID()) {
			targetMainFork = target
		} else {
			targetMainFork = bd.getMainFork(target, true)
		}
	}

	if targetMainFork == nil {
		return false, targetMainFork
	}
	if int64(mainTip.GetLayer())-int64(targetMainFork.GetLayer()) >= int64(max) {
		return bd.instance.(*Phantom).doIsBlue(target, targetMainFork), targetMainFork
	}
	//
	queueSet := NewIdSet()
	queue := []IBlock{}
	queue = append(queue, mainTip)
	queueSet.Add(mainTip.GetID())

	connected := false
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.GetID() == target.GetID() {
			connected = true
			break
		}
		if !cur.HasParents() {
			continue
		}
		if cur.GetLayer() <= target.GetLayer() {
			continue
		}

		for _, v := range bd.getParents(cur).GetMap() {
			ib := v.(IBlock)
			if queueSet.Has(ib.GetID()) {
				continue
			}
			queue = append(queue, ib)
			queueSet.Add(ib.GetID())
		}
	}
	if connected {
		return bd.instance.(*Phantom).doIsBlue(target, targetMainFork), targetMainFork
	}
	return connected, targetMainFork
}
