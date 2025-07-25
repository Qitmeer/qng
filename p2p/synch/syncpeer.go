package synch

import (
	"github.com/Qitmeer/qng/v2/p2p/peers"
	"github.com/libp2p/go-libp2p/core/peer"
	"math/rand"
)

func (ps *PeerSync) HasSyncPeer() bool {
	return ps.SyncPeer() != nil
}

func (ps *PeerSync) SyncPeer() *peers.Peer {
	ps.splock.RLock()
	defer ps.splock.RUnlock()

	return ps.syncPeer
}

func (ps *PeerSync) SetSyncPeer(pe *peers.Peer) {
	ps.splock.Lock()
	defer ps.splock.Unlock()

	ps.syncPeer = pe
}

func (ps *PeerSync) isSyncPeer(pe *peers.Peer) bool {
	ps.splock.RLock()
	defer ps.splock.RUnlock()

	if ps.syncPeer == nil || pe == nil {
		return false
	}
	if pe == ps.syncPeer || pe.GetID() == ps.syncPeer.GetID() {
		return true
	}
	return false
}

// getBestPeer
func (ps *PeerSync) getBestPeer(sm SyncMode, exclude map[peer.ID]struct{}) *peers.Peer {
	best := ps.Chain().BestSnapshot()
	var bestPeer *peers.Peer
	equalPeers := []*peers.Peer{}
	for _, sp := range ps.sy.peers.CanSyncPeers() {
		if !ps.isValidSyncPeer(sp, sm) {
			continue
		}
		if sm.IsSnap() {
			if ps.IsSnapSync() && sp.GetID() == ps.snapStatus.PeerID() {
				return sp
			}
		}
		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While techcnically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		gs := sp.GraphState()
		if gs == nil {
			continue
		}
		if !gs.IsExcellent(best.GraphState) {
			continue
		}
		// the best sync candidate is the most updated peer
		if bestPeer == nil {
			bestPeer = sp
			equalPeers = []*peers.Peer{sp}
			continue
		}
		if gs.IsExcellent(bestPeer.GraphState()) {
			bestPeer = sp
			equalPeers = []*peers.Peer{sp}
		} else if gs.IsEqual(bestPeer.GraphState()) {
			equalPeers = append(equalPeers, sp)
		}
	}
	if len(equalPeers) <= 0 {
		return nil
	}
	if len(equalPeers) == 1 {
		return equalPeers[0]
	}
	if len(exclude) > 0 {
		filter := []*peers.Peer{}
		for _, sp := range equalPeers {
			_, ok := exclude[sp.GetID()]
			if ok {
				continue
			}
			filter = append(filter, sp)
		}
		if len(filter) == 1 {
			return filter[0]
		} else if len(filter) > 1 {
			equalPeers = filter
		}
	}
	for _, sp := range equalPeers {
		if sp.IsSupportSyncBlocksLong() {
			return sp
		}
	}
	index := int(rand.Int63n(int64(len(equalPeers))))
	if index >= len(equalPeers) {
		index = 0
	}
	return equalPeers[index]
}

func (ps *PeerSync) IsCompleteForSyncPeer() bool {
	// if blockChain thinks we are current and we have no syncPeer it
	// is probably right.
	sp := ps.SyncPeer()
	if sp == nil {
		return true
	}

	// No matter what chain thinks, if we are below the block we are syncing
	// to we are not current.
	gs := sp.GraphState()
	if gs == nil {
		return true
	}
	if gs.IsExcellent(ps.Chain().BestSnapshot().GraphState) {
		//log.Trace("comparing the current best vs sync last",
		//	"current.best", ps.Chain().BestSnapshot().GraphState.String(), "sync.last", gs.String())
		return false
	}

	return true
}

func (ps *PeerSync) isValidSyncPeer(pe *peers.Peer, sm SyncMode) bool {
	if !ps.sy.peers.IsActive(pe) ||
		!pe.IsConsensus() ||
		pe.IsSnapSync() {
		return false
	}
	if sm.IsSnap() {
		if !pe.IsSnap() || pe.GetMeerState() == nil || !pe.GetMeerConn() || !pe.IsLongConn() {
			return false
		}
		if pe.GetMeerState().Number <= MinSnapSyncNumber {
			return false
		}
	} else if sm.IsLong() {
		if !pe.IsSupportSyncBlocksLong() || !pe.IsLongConn() {
			return false
		}
	}
	return true
}
