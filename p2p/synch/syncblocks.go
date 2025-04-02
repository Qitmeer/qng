/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"time"
)

func (ps *PeerSync) syncBlocks(pe *peers.Peer) int {
	getSyncPeer := func() *peers.Peer {
		start := time.Now()
		var pee *peers.Peer
		for pee == nil {
			pee = ps.getBestPeer(LongSyncMode, nil)
			if pee == nil {
				log.Debug("Try to get sync peer", "cost", time.Since(start).String())
				time.Sleep(time.Second * 5)
			}
			if !ps.IsRunning() {
				return nil
			}
		}
		return pee
	}
	behaviorFlags := blockchain.BFP2PAdd
	add := 0
	init := true
	zeroGS := &pb.GraphState{Tips: []*pb.Hash{&pb.Hash{Hash: hash.ZeroHash.Bytes()}}}
	var gs *pb.GraphState
	var mainLocator []*hash.Hash
	var latest *hash.Hash
	for pe != nil {
		if ps.IsInterrupt() {
			break
		}
		point := pe.SyncPoint()
		if init {
			gs = ps.sy.getGraphState()
			mainLocator, _ = ps.dagSync.GetMainLocator(point, false)
		} else {
			gs = zeroGS
			mainLocator = []*hash.Hash{point, latest}
		}
		req := &pb.SyncBlocksReq{Init: init, GraphState: gs, MainLocator: changeHashsToPBHashs(mainLocator)}
		ret, err := pe.Request(SyncBlocksReqMsg, req)
		if err != nil {
			log.Error(err.Error())
			if !ps.isValidSyncPeer(pe, LongSyncMode) {
				pe = getSyncPeer()
			} else {
				break
			}
			init = true
			continue
		}
		if ret != nil {
			init = false
			rsp, ok := ret.(*pb.SyncBlocksRsp)
			if !ok {
				log.Error("Response message type is not SyncBlocksRsp")
				break
			}
			log.Debug("Receive blocks", "count", len(rsp.Blocks), "point", changePBHashToHash(rsp.SyncPoint).String())
			if isZeroPBHash(rsp.SyncPoint) {
				break
			}
			if len(rsp.Blocks) > 0 {
				for _, bs := range rsp.Blocks {
					block, err := types.NewBlockFromBytes(bs.Data)
					if err != nil {
						log.Warn("NewBlockFromBytes", "err", err, "processID", ps.getProcessID())
						init = true
						break
					}
					if ps.sy.p2p.BlockChain().BlockDAG().HasBlock(block.Hash()) {
						continue
					}
					pid := pe.GetID()
					_, IsOrphan, err := ps.sy.p2p.BlockChain().ProcessBlock(block, behaviorFlags, &pid)
					if err != nil {
						log.Error("Failed to process block", "hash", block.Hash(), "err", err, "processID", ps.getProcessID())
						init = true
						break
					}
					if IsOrphan {
						init = true
						break
					}
					add++
					latest = block.Hash()
				}
			} else {
				break
			}
			if rsp.Complete {
				break
			}
		}
	}
	return add
}

func (s *Sync) syncBlocksHandler(id uint64, msg interface{}, pe *peers.Peer) error {
	m, ok := msg.(*pb.SyncBlocksReq)
	if !ok {
		return fmt.Errorf("message is not type *SyncBlocksReq")
	}
	locator := changePBHashsToHashs(m.MainLocator)
	bd := s.p2p.BlockChain().BlockDAG()
	rsp := &pb.SyncBlocksRsp{
		SyncPoint: &pb.Hash{Hash: hash.ZeroHash.Bytes()},
		Blocks:    []*pb.BytesData{},
		Complete:  true,
	}
	if m.Init {
		pe.UpdateGraphState(m.GraphState)
	}
	var startBlock meerdag.IBlock
	var point meerdag.IBlock
	if m.Init {
		for i := len(locator) - 1; i >= 0; i-- {
			mainBlock := bd.GetBlock(locator[i])
			if mainBlock == nil {
				continue
			}
			if !bd.IsOnMainChain(mainBlock.GetID()) {
				continue
			}
			point = mainBlock
			break
		}

		if point == nil && len(locator) > 0 {
			point = bd.GetBlock(locator[0])
			if point != nil {
				for !bd.IsOnMainChain(point.GetID()) {
					if point.GetMainParent() == meerdag.MaxId {
						break
					}
					point = bd.GetBlockById(point.GetMainParent())
					if point == nil {
						break
					}
				}
			}
		}
		if point == nil {
			point = bd.GetGenesis()
		}
		startBlock = point
	} else {
		if len(locator) == 2 {
			point = bd.GetBlock(locator[0])
			if point != nil {
				if bd.IsOnMainChain(point.GetID()) {
					startBlock = bd.GetBlock(locator[1])
				}
			}
		}
	}
	if startBlock != nil {
		rsp.SyncPoint = &pb.Hash{Hash: point.GetHash().Bytes()}
		rsp.Complete = s.PeerSync().dagSync.TraverseBlocks(startBlock, func(block meerdag.IBlock) bool {
			bs, err := s.p2p.BlockChain().FetchBlockBytesByHash(block.GetHash())
			if err != nil {
				log.Error(err.Error())
				return false
			}
			pbbd := &pb.BytesData{Data: bs}
			if uint64(rsp.SizeSSZ()+pbbd.SizeSSZ()+BLOCKDATA_SSZ_HEAD_SIZE) >= s.p2p.Encoding().GetMaxChunkSize() {
				return false
			}
			rsp.Blocks = append(rsp.Blocks, pbbd)
			return true
		})
	}
	return pe.Respond(SyncBlocksRspMsg, rsp, id)
}
