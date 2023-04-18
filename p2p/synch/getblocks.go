/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (s *Sync) sendGetBlocksRequest(ctx context.Context, id peer.ID, blocks *pb.GetBlocks) (*pb.DagBlocks, error) {
	ctx, cancel := context.WithTimeout(ctx, ReqTimeout)
	defer cancel()

	stream, err := s.Send(ctx, blocks, RPCGetBlocks, id)
	if err != nil {
		return nil, err
	}

	code, errMsg, err := ReadRspCode(stream, s.p2p)
	if err != nil {
		return nil, err
	}

	if !code.IsSuccess() {
		s.Peers().IncrementBadResponses(stream.Conn().RemotePeer(), "get blocks request rsp")
		closeStream(stream, s.p2p)
		return nil, errors.New(errMsg)
	}

	msg := &pb.DagBlocks{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, err
	}
	closeStream(stream, s.p2p)
	return msg, err
}

func (s *Sync) getBlocksHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) *common.Error {
	ctx, cancel := context.WithTimeout(ctx, HandleTimeout)
	var err error
	defer cancel()

	m, ok := msg.(*pb.GetBlocks)
	if !ok {
		err = fmt.Errorf("message is not type *pb.Hash")
		return ErrMessage(err)
	}
	blocks, _ := s.PeerSync().dagSync.CalcSyncBlocks(nil, changePBHashsToHashs(m.Locator), meerdag.DirectMode, MaxBlockLocatorsPerMsg)
	bd := &pb.DagBlocks{Blocks: changeHashsToPBHashs(blocks)}
	e := s.EncodeResponseMsg(stream, bd)
	if e != nil {
		return e
	}
	return nil
}

func (ps *PeerSync) processGetBlocks(pe *peers.Peer, blocks []*hash.Hash) *ProcessResult {
	db, err := ps.sy.sendGetBlocksRequest(ps.sy.p2p.Context(), pe.GetID(), &pb.GetBlocks{Locator: changeHashsToPBHashs(blocks)})
	if err != nil {
		log.Warn(err.Error(), "processID", ps.processID)
		return nil
	}
	if len(db.Blocks) <= 0 {
		log.Warn("no block need to get", "processID", ps.processID)
		return nil
	}
	if ps.IsInterrupt() {
		return nil
	}
	return ps.processGetBlockDatas(pe, changePBHashsToHashs(db.Blocks))
}

func (s *Sync) GetDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) *common.Error {
	ctx, cancel := context.WithTimeout(ctx, HandleTimeout)
	var err error
	defer func() {
		cancel()
	}()
	pe := s.peers.Get(stream.Conn().RemotePeer())
	if pe == nil {
		return ErrPeerUnknown
	}
	m, ok := msg.(*pb.Inventory)
	if !ok {
		err = fmt.Errorf("message is not type *MsgFilterLoad")
		return ErrMessage(err)
	}
	s.peerSync.msgChan <- &GetDatasMsg{pe: pe, data: m}
	return nil
}
