package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"sync"
	"sync/atomic"
	"time"
)

// Check data consistency
func (s *Sync) CheckConsistency(hashOrOrder *protocol.HashOrNumber) (string, error) {
	if !s.p2p.Config().Consistency {
		return "", fmt.Errorf("Please enable --consistency")
	}
	var stateRoot hash.Hash
	var block meerdag.IBlock
	if hashOrOrder == nil {
		bs := s.p2p.BlockChain().BestSnapshot()
		block = s.p2p.BlockChain().BlockDAG().GetBlock(&bs.Hash)
		stateRoot = bs.StateRoot
	} else {
		if hashOrOrder.IsHash() {
			block = s.p2p.BlockChain().BlockDAG().GetBlock(hashOrOrder.Hash)
		} else {
			block = s.p2p.BlockChain().BlockDAG().GetBlockByOrder(uint(hashOrOrder.Number))
		}
		if block == nil {
			return "", fmt.Errorf("No block:%v\n", hashOrOrder)
		}
		stateRoot = *block.GetState().Root()
	}

	startTime := time.Now()

	connectedPeers := s.Peers().CanSyncPeers()

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
	if len(pes) <= 0 {
		return "", fmt.Errorf("No peers")
	}

	total := int32(0)
	valid := int32(0)
	wg := sync.WaitGroup{}

	log.Info("Start to check data consistency by p2p", "hash", block.GetHash().String(), "order", block.GetOrder(), "stateRoot", stateRoot.String())
	for _, pe := range pes {
		wg.Add(1)
		go func(pee *peers.Peer) {
			log.Info("Data consistency start", "block", block.GetHash().String(), "state root", stateRoot.String(), "peer", pee.IDWithAddress())

			root, err := s.Send(pee, RPCStateRoot, &pb.StateRootReq{Block: &pb.Hash{Hash: block.GetHash().Bytes()}})
			if err == nil && root != nil {
				sr, ok := root.(*hash.Hash)
				if ok && sr != nil {
					atomic.AddInt32(&total, 1)
					if sr.IsEqual(&stateRoot) {
						atomic.AddInt32(&valid, 1)
					} else {
						log.Info("Data inconsistency", "block", block.GetHash().String(), "stateRoot", stateRoot.String(), "peerStateRoot", sr.String(), "peer", pee.IDWithAddress())
					}
				}
			}
			wg.Done()
		}(pe)
	}
	wg.Wait()
	if total <= 0 {
		return "No suitable results", nil
	}
	ratio := float64(valid) / float64(total)
	if ratio < 0.5 {
		// process
		return fmt.Sprintf("Data inconsistency: ratio=%f,block=%s,stateRoot=%s,elapsed=%s,height=%d", ratio, block.GetHash().String(), stateRoot.String(), time.Since(startTime).String(), block.GetHeight()), nil
	} else {
		return fmt.Sprintf("Data consistency: ratio=%f,block=%s,stateRoot=%s,elapsed=%s,height=%d", ratio, block.GetHash().String(), stateRoot.String(), time.Since(startTime).String(), block.GetHeight()), nil
	}
}
