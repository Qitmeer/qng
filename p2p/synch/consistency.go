package synch

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/runutil"
	"github.com/Qitmeer/qng/params"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MinConsistencyPeerNum = 5
)

// Check data consistency
func (s *Sync) consistency() {
	if !s.p2p.Config().Consistency {
		return
	}
	log.Info("Enable check data consistency by p2p")
	runutil.RunEvery(s.p2p.Context(), params.ActiveNetParams.TargetTimePerBlock*meerdag.StableConfirmations, func() {
		startTime := time.Now()
		mt := s.p2p.BlockChain().BlockDAG().GetMainChainTip()
		block := s.p2p.BlockChain().BlockDAG().RelativeMainAncestor(mt, meerdag.StableConfirmations)
		if block == nil || block.GetState() == nil {
			return
		}
		connectedPeers := s.Peers().CanSyncPeers()
		if len(connectedPeers) <= MinConsistencyPeerNum {
			return
		}
		pes := []*peers.Peer{}
		for _, pe := range connectedPeers {
			if pe.ChainState() == nil {
				continue
			}
			if pe.ChainState().ProtocolVersion < protocol.ProtocolVersion {
				continue
			}
			pes = append(pes, pe)
		}
		if len(pes) <= MinConsistencyPeerNum {
			return
		}
		stateRoot := *block.GetState().Root()
		total := int32(0)
		valid := int32(0)
		wg := sync.WaitGroup{}
		for _, pe := range pes {
			wg.Add(1)
			go func(pee *peers.Peer) {
				log.Debug("Data consistency start", "block", block.GetHash().String(), "state root", stateRoot.String(), "peer", pee.IDWithAddress())

				root, err := s.Send(pee, RPCStateRoot, &pb.StateRootReq{Block: &pb.Hash{Hash: block.GetHash().Bytes()}})
				if err == nil {
					atomic.AddInt32(&total, 1)
					if root != nil {
						sr, ok := root.(*hash.Hash)
						if ok {
							if sr.IsEqual(&stateRoot) {
								atomic.AddInt32(&valid, 1)
							}
						}
					}
				}
				wg.Done()
			}(pe)
		}
		wg.Wait()
		if total <= MinConsistencyPeerNum {
			return
		}
		ratio := float64(valid) / float64(total)
		if ratio < 0.5 {
			// process
			log.Error("Data inconsistency ", "ratio", ratio, "block", block.GetHash().String(), "state root", stateRoot.String(), "elapsed", time.Since(startTime).String(), "height", block.GetHeight())
			s.p2p.Consensus().Shutdown()
		} else {
			log.Debug("Data consistency end", "ratio", ratio, "block", block.GetHash().String(), "state root", stateRoot.String(), "elapsed", time.Since(startTime).String(), "height", block.GetHeight())
		}
	})
}
