/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

func (s *Sync) sendStateRootRequest(stream network.Stream, pe *peers.Peer) (*hash.Hash, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("state root request rsp")
		return nil, e
	}
	msg := &pb.StateRootRsp{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	if !msg.Has {
		return nil,nil
	}
	return changePBHashToHash(msg.Root), nil
}

func (s *Sync) stateRootHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.StateRootReq)
	if !ok {
		err := fmt.Errorf("message is not type *pb.StateRootReq")
		return ErrMessage(err)
	}
	blockHash:=changePBHashToHash(m.Block)
	if blockHash == nil {
		return ErrMessage(fmt.Errorf("invalid block hash"))
	}
	rsp:=&pb.StateRootRsp{Has: false}
	block:=s.p2p.BlockChain().BlockDAG().GetBlock(blockHash)
	if block != nil && block.GetState() != nil {
		rsp.Has=true
		rsp.Root=&pb.Hash{Hash: block.GetState().Root().Bytes()}
	}
	return s.EncodeResponseMsg(stream, rsp)
}
