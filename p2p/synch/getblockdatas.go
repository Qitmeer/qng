/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/bloom"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

const BLOCKDATA_SSZ_HEAD_SIZE = 4

func (s *Sync) sendGetBlockDataRequest(stream network.Stream, pe *peers.Peer) (*pb.BlockDatas, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("get block date request rsp")
		return nil, e
	}
	msg := &pb.BlockDatas{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}

func (s *Sync) sendGetMerkleBlockDataRequest(ctx context.Context, id peer.ID, req *pb.MerkleBlockRequest) (*pb.MerkleBlockResponse, error) {
	return nil, nil
}

func (s *Sync) getBlockDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.GetBlockDatas)
	if !ok {
		err := fmt.Errorf("message is not type *pb.Hash")
		return ErrMessage(err)
	}
	bds := []*pb.BlockData{}
	bd := &pb.BlockDatas{Locator: bds}
	for _, bdh := range m.Locator {
		blockHash, err := hash.NewHash(bdh.Hash)
		if err != nil {
			err = fmt.Errorf("invalid block hash")
			return ErrMessage(err)
		}
		blocks, err := s.p2p.BlockChain().FetchBlockBytesByHash(blockHash)
		if err != nil {
			return ErrMessage(err)
		}
		pbbd := pb.BlockData{BlockBytes: blocks}
		if uint64(bd.SizeSSZ()+pbbd.SizeSSZ()+BLOCKDATA_SSZ_HEAD_SIZE) >= s.p2p.Encoding().GetMaxChunkSize() {
			break
		}
		bd.Locator = append(bd.Locator, &pbbd)
	}
	return s.EncodeResponseMsg(stream, bd)
}

func (s *Sync) getMerkleBlockDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {

	m, ok := msg.(*pb.MerkleBlockRequest)
	if !ok {
		err := fmt.Errorf("message is not type *pb.Hash")
		return ErrMessage(err)
	}
	filter := pe.Filter()
	// Do not send a response if the peer doesn't have a filter loaded.
	if !filter.IsLoaded() {
		log.Warn("filter not loaded!")
		return nil
	}
	bds := []*pb.MerkleBlock{}
	bd := &pb.MerkleBlockResponse{Data: bds}
	for _, bdh := range m.Hashes {
		blockHash, err := hash.NewHash(bdh.Hash)
		if err != nil {
			err = fmt.Errorf("invalid block hash")
			return ErrMessage(err)
		}
		block, err := s.p2p.BlockChain().FetchBlockByHash(blockHash)
		if err != nil {
			return ErrMessage(err)
		}
		// Generate a merkle block by filtering the requested block according
		// to the filter for the peer.
		merkle, _ := bloom.NewMerkleBlock(block, filter)
		// Finally, send any matched transactions.
		pbbd := pb.MerkleBlock{Header: merkle.Header.BlockData(),
			Transactions: uint64(merkle.Transactions),
			Hashes:       changeHashsToPBHashs(merkle.Hashes),
			Flags:        merkle.Flags,
		}
		bd.Data = append(bd.Data, &pbbd)
	}
	e := s.EncodeResponseMsg(stream, bd)
	if e != nil {
		return e
	}
	return nil
}

func (ps *PeerSync) processGetBlockDatas(pe *peers.Peer, blocks []*hash.Hash) *ProcessResult {
	if !ps.isSyncPeer(pe) || !pe.IsConnected() {
		err := fmt.Errorf("no sync peer:%v", pe.GetID())
		log.Trace(err.Error(), "processID", ps.processID)
		return nil
	}
	blocksReady := []*hash.Hash{}
	blockDatas := []*BlockData{}
	blockDataM := map[hash.Hash]*BlockData{}

	for _, b := range blocks {
		if ps.IsInterrupt() {
			return nil
		}
		if ps.sy.p2p.BlockChain().BlockDAG().HasBlock(b) {
			continue
		}
		blkd := &BlockData{Hash: b}
		blockDataM[*blkd.Hash] = blkd
		blockDatas = append(blockDatas, blkd)
		if ps.sy.p2p.BlockChain().IsOrphan(b) {
			ob := ps.sy.p2p.BlockChain().GetOrphan(b)
			if ob != nil {
				blkd.Block = ob
				continue
			}
		} else if ps.sy.p2p.BlockChain().HasBlockInDB(b) {
			sb, err := ps.sy.p2p.BlockChain().FetchBlockByHash(b)
			if err == nil {
				blkd.Block = sb
				continue
			}
		}
		blocksReady = append(blocksReady, b)
	}
	if len(blockDatas) <= 0 {
		return &ProcessResult{act: ProcessResultActionContinue, orphan: false}
	}
	readys := len(blocksReady)
	packageNumber := 0
	index := 0
	getBlockDatas := func() bool {
		if readys > 0 && index < readys {
			packageNumber++
			sendBlocks := blocksReady[index:]
			log.Trace(fmt.Sprintf("processGetBlockDatas::sendGetBlockDataRequest peer=%v, blocks=%d [%s -> %s] ", pe.GetID(), len(sendBlocks), sendBlocks[0], sendBlocks[len(sendBlocks)-1]), "processID", ps.processID, "package number", packageNumber)
			ret, err := ps.sy.Send(pe, RPCGetBlockDatas, &pb.GetBlockDatas{Locator: changeHashsToPBHashs(sendBlocks)})
			if err != nil {
				log.Warn(fmt.Sprintf("getBlocks send:%v", err), "processID", ps.processID)
				if index == 0 {
					index++
					return false
				} else {
					index = readys
					return true
				}
			}
			bd := ret.(*pb.BlockDatas)
			log.Trace(fmt.Sprintf("Received:Locator=%d", len(bd.Locator)), "processID", ps.processID)
			//
			var lastBlockHash *hash.Hash
			for i, b := range bd.Locator {
				block, err := types.NewBlockFromBytes(b.BlockBytes)
				if err != nil {
					log.Warn(fmt.Sprintf("getBlocks from:%v", err), "processID", ps.processID)
					break
				}
				bdm, ok := blockDataM[*block.Hash()]
				if ok {
					bdm.Block = block
				}
				if i+1 == len(bd.Locator) {
					lastBlockHash = block.Hash()
				}
			}
			if lastBlockHash != nil {
				index++
				for i := 0; i < readys; i++ {
					if lastBlockHash.IsEqual(blocksReady[i]) {
						index = i + 1
						return true
					}
				}
			} else {
				index = readys
			}
			return true
		}
		return true
	}

	behaviorFlags := blockchain.BFP2PAdd
	add := 0
	hasOrphan := false

	for i, b := range blockDatas {
		if ps.IsInterrupt() ||
			!ps.IsRunning() {
			return nil
		}
		block := b.Block
		if block == nil {
			ret := getBlockDatas()
			if !ret {
				return &ProcessResult{act: ProcessResultActionTryAgain}
			}
			if b.Block == nil {
				log.Trace(fmt.Sprintf("No block bytes:%d : %s", i, b.Hash.String()), "processID", ps.processID)
				continue
			}
			block = b.Block
		}
		//
		IsOrphan, err := ps.sy.p2p.BlockChain().ProcessBlock(block, behaviorFlags)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to process block:hash=%s err=%s", block.Hash(), err), "processID", ps.processID)
			continue
		}
		if IsOrphan {
			hasOrphan = true
			continue
		}
		ps.sy.p2p.RegainMempool()

		add++
		ps.lastSync = time.Now()
	}
	log.Debug(fmt.Sprintf("getBlockDatas:%d/%d", add, len(blockDatas)), "processID", ps.processID)

	var err error
	if add > 0 {
		ps.sy.p2p.TxMemPool().PruneExpiredTx()
	} else {
		err = fmt.Errorf("no get blocks")
		log.Debug(err.Error(), "processID", ps.processID)
		return &ProcessResult{act: ProcessResultActionTryAgain}
	}
	return &ProcessResult{act: ProcessResultActionContinue, orphan: hasOrphan, add: add}
}

func (ps *PeerSync) processGetMerkleBlockDatas(pe *peers.Peer, blocks []*hash.Hash) error {
	if !ps.isSyncPeer(pe) || !pe.IsConnected() {
		err := fmt.Errorf("no sync peer")
		log.Trace(err.Error())
		return err
	}
	filter := pe.Filter()
	// Do not send a response if the peer doesn't have a filter loaded.
	if !filter.IsLoaded() {
		err := fmt.Errorf("filter not loaded")
		log.Trace(err.Error())
		return nil
	}

	blocksReady := []*hash.Hash{}

	for _, b := range blocks {
		if ps.sy.p2p.BlockChain().HaveBlock(b) {
			continue
		}
		blocksReady = append(blocksReady, b)
	}
	if len(blocksReady) <= 0 {
		return nil
	}

	bd, err := ps.sy.sendGetMerkleBlockDataRequest(ps.sy.p2p.Context(), pe.GetID(), &pb.MerkleBlockRequest{Hashes: changeHashsToPBHashs(blocksReady)})
	if err != nil {
		log.Warn(fmt.Sprintf("sendGetMerkleBlockDataRequest send:%v", err))
		return err
	}
	log.Debug(fmt.Sprintf("sendGetMerkleBlockDataRequest:%d", len(bd.Data)))
	return nil
}

// handleGetData is invoked when a peer receives a getdata qitmeer message and
// is used to deliver block and transaction information.
func (ps *PeerSync) OnGetData(sp *peers.Peer, invList []*pb.InvVect) error {
	txs := make([]*pb.Hash, 0)
	blocks := make([]*pb.Hash, 0)
	merkleBlocks := make([]*pb.Hash, 0)
	for _, iv := range invList {
		log.Trace(fmt.Sprintf("OnGetData:%s (%s)", InvType(iv.Type).String(), changePBHashToHash(iv.Hash)))
		switch InvType(iv.Type) {
		case InvTypeTx:
			txs = append(txs, iv.Hash)
		case InvTypeBlock:
			blocks = append(blocks, iv.Hash)
		case InvTypeFilteredBlock:
			merkleBlocks = append(merkleBlocks, iv.Hash)
		default:
			log.Warn(fmt.Sprintf("Unknown type in inventory request %d",
				iv.Type))
			continue
		}
	}
	if len(txs) > 0 {
		err := ps.processGetTxs(sp, changePBHashsToHashs(txs))
		if err != nil {
			log.Info("processGetTxs Error", "err", err.Error())
			return err
		}
	}
	if len(blocks) > 0 {
		ps.processGetBlockDatas(sp, changePBHashsToHashs(blocks))
	}
	if len(merkleBlocks) > 0 {
		err := ps.processGetMerkleBlockDatas(sp, changePBHashsToHashs(merkleBlocks))
		if err != nil {
			log.Info("processGetBlockDatas Error", "err", err.Error())
			return err
		}
	}
	return nil
}
