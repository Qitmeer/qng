/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

func (s *Sync) sendGetBlocksRequest(stream network.Stream, pe *peers.Peer) (*pb.DagBlocks, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("get blocks request rsp")
		return nil, e
	}
	msg := &pb.DagBlocks{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}

func (s *Sync) getBlocksHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.GetBlocks)
	if !ok {
		err := fmt.Errorf("message is not type *pb.Hash")
		return ErrMessage(err)
	}
	blocks, _ := s.PeerSync().dagSync.CalcSyncBlocks(nil, changePBHashsToHashs(m.Locator), meerdag.DirectMode, MaxBlockLocatorsPerMsg)
	bd := &pb.DagBlocks{Blocks: changeHashsToPBHashs(blocks)}
	return s.EncodeResponseMsg(stream, bd)
}

func (ps *PeerSync) processGetBlocks(pe *peers.Peer, blocks []*hash.Hash) *ProcessResult {
	ret, err := ps.sy.Send(pe, RPCGetBlocks, &pb.GetBlocks{Locator: changeHashsToPBHashs(blocks)})
	if err != nil {
		log.Warn(err.Error(), "processID", ps.processID)
		return nil
	}
	db := ret.(*pb.DagBlocks)
	if len(db.Blocks) <= 0 {
		log.Warn("no block need to get", "processID", ps.processID)
		return nil
	}
	if ps.IsInterrupt() {
		return nil
	}
	return ps.processGetBlockDatas(pe, changePBHashsToHashs(db.Blocks))
}

func (s *Sync) GetDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.Inventory)
	if !ok {
		err := fmt.Errorf("message is not type *MsgFilterLoad")
		return ErrMessage(err)
	}
	s.peerSync.msgChan <- &GetDatasMsg{pe: pe, data: m}
	return nil
}
