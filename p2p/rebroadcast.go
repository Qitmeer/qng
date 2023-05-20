package p2p

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/synch"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/notifymgr/notify"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type broadcastInventoryAdd relayMsg

type broadcastInventoryDel *hash.Hash

type relayMsg struct {
	hash *hash.Hash
	data interface{}
}

type Rebroadcast struct {
	started  int32
	shutdown int32

	wg   sync.WaitGroup
	quit chan struct{}

	modifyRebroadcastInv chan interface{}

	s *Service

	regainMP      bool
	regainMPLimit int
}

func (r *Rebroadcast) Start() {
	// Already started?
	if atomic.AddInt32(&r.started, 1) != 1 {
		return
	}

	log.Info("Starting Rebroadcast")

	r.wg.Add(2)
	go r.handler()
	go r.mempoolHandler()
}

func (r *Rebroadcast) Stop() error {
	// Make sure this only happens once.
	if atomic.AddInt32(&r.shutdown, 1) != 1 {
		log.Info("Rebroadcast is already in the process of shutting down")
		return nil
	}

	log.Info("Rebroadcast shutting down")

	close(r.quit)

	r.wg.Wait()
	return nil

}

func (r *Rebroadcast) mempoolHandler() {
	timer := time.NewTicker(params.ActiveNetParams.TargetTimePerBlock)
out:
	for {
		select {
		case <-timer.C:
			r.onRegainMempool()
		case <-r.quit:
			break out
		}
	}
	timer.Stop()
	r.wg.Done()
}

func (r *Rebroadcast) handler() {
	timer := time.NewTimer(params.ActiveNetParams.TargetTimePerBlock)
	pendingInvs := make(map[hash.Hash]interface{})

out:
	for {
		select {
		case riv := <-r.modifyRebroadcastInv:
			switch msg := riv.(type) {
			case broadcastInventoryAdd:
				pendingInvs[*msg.hash] = msg.data
			case broadcastInventoryDel:
				delete(pendingInvs, *msg)
			}

		case <-timer.C:
			isCurrent := r.s.PeerSync().IsCurrent()
			nds := []*notify.NotifyData{}
			for h, data := range pendingInvs {
				dh := h
				if _, ok := data.(*types.TxDesc); ok {
					if !r.s.TxMemPool().HaveTransaction(&dh) {
						delete(pendingInvs, dh)
						continue
					}
					if !isCurrent {
						continue
					}
				}
				nds = append(nds, &notify.NotifyData{Data: data})
			}

			if len(nds) > 0 {
				r.s.RelayInventory(nds)
			}

			rt := int64(len(pendingInvs)/50) * int64(params.ActiveNetParams.TargetTimePerBlock)
			if rt < int64(params.ActiveNetParams.TargetTimePerBlock) {
				rt = int64(params.ActiveNetParams.TargetTimePerBlock)
			}
			timer.Reset(time.Duration(rt))

			r.s.sy.Peers().UpdateBroadcasts()
		case <-r.quit:
			break out
		}
	}
	timer.Stop()

cleanup:
	for {
		select {
		case <-r.modifyRebroadcastInv:
		default:
			break cleanup
		}
	}
	r.wg.Done()
}

func (r *Rebroadcast) AddInventory(h *hash.Hash, data interface{}) {
	// Ignore if shutting down.
	if atomic.LoadInt32(&r.shutdown) != 0 {
		return
	}

	r.modifyRebroadcastInv <- broadcastInventoryAdd{hash: h, data: data}
}

func (r *Rebroadcast) RemoveInventory(h *hash.Hash) {
	// Ignore if shutting down.
	if atomic.LoadInt32(&r.shutdown) != 0 {
		return
	}

	r.modifyRebroadcastInv <- broadcastInventoryDel(h)
}

func (r *Rebroadcast) RegainMempool() {
	if r.regainMP {
		return
	}
	r.regainMP = true
}

func (r *Rebroadcast) onRegainMempool() {
	if !r.s.PeerSync().IsCurrent() {
		return
	}
	mptxCount := r.s.TxMemPool().Count()

	canPeers := []*peers.Peer{}
	for _, pe := range r.s.Peers().CanSyncPeers() {
		if time.Since(pe.GetMempoolReqTime()) <= params.ActiveNetParams.TargetTimePerBlock {
			continue
		}
		canPeers = append(canPeers, pe)
	}
	if len(canPeers) <= 0 {
		return
	}
	index := rand.Intn(len(canPeers))
	pe := canPeers[index]
	go r.s.sy.Send(pe, synch.RPCMemPool, &pb.MemPoolRequest{TxsNum: uint64(mptxCount)})
}

func NewRebroadcast(s *Service) *Rebroadcast {
	r := Rebroadcast{
		s:                    s,
		quit:                 make(chan struct{}),
		modifyRebroadcastInv: make(chan interface{}),
		regainMP:             true,
		regainMPLimit:        1,
	}

	return &r
}
