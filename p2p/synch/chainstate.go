/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/v2/common/roughtime"
	"github.com/Qitmeer/qng/v2/p2p/common"
	"github.com/Qitmeer/qng/v2/p2p/peers"
	pb "github.com/Qitmeer/qng/v2/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	MaxPBGraphStateTips = 100
)

func (s *Sync) sendChainStateRequest(stream network.Stream, pe *peers.Peer) *common.Error {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() && !e.Code.IsDAGConsensus() {
		e.Add("chain state request")
		return e
	}
	msg := &pb.ChainState{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return common.NewError(common.ErrStreamRead, err)
	}
	s.UpdateChainState(pe, ChangeChainStateV1ToV2(msg), !e.Code.IsDAGConsensus())
	if !e.Code.IsSuccess() {
		return e
	}
	e = s.validateChainStateMessage(ChangeChainStateV1ToV2(msg), pe)
	if e != nil {
		if e.Code.IsDAGConsensus() {
			go s.sendGoodByeAndDisconnect(common.ErrDAGConsensus, pe)
		}
		return e
	}
	return e
}

func (s *Sync) chainStateHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.ChainState)
	if !ok {
		return ErrMessage(fmt.Errorf("message is not type *pb.ChainState"))
	}
	e := s.validateChainStateMessage(ChangeChainStateV1ToV2(m), pe)
	if e != nil {
		if e.Code.IsDAGConsensus() {
			// Respond with our status and disconnect with the peer.
			s.UpdateChainState(pe, ChangeChainStateV1ToV2(m), false)
			if err := s.EncodeResponseMsgPro(stream, s.getChainState(), e.Code); err != nil {
				return err
			}
		}
		return e
	}
	if !s.bidirectionalChannelCapacity(pe, stream.Conn()) {
		s.UpdateChainState(pe, ChangeChainStateV1ToV2(m), false)
		if err := s.EncodeResponseMsgPro(stream, s.getChainState(), common.ErrDAGConsensus); err != nil {
			return err
		}
		return ErrMessage(fmt.Errorf("bidirectional channel capacity"))
	}
	s.UpdateChainState(pe, ChangeChainStateV1ToV2(m), true)
	return s.EncodeResponseMsg(stream, s.getChainState())
}

func (s *Sync) getChainState() *pb.ChainState {
	genesisHash := s.p2p.GetGenesisHash()

	cs := &pb.ChainState{
		GenesisHash:     &pb.Hash{Hash: genesisHash.Bytes()},
		ProtocolVersion: s.p2p.Config().ProtocolVersion,
		Timestamp:       uint64(roughtime.Now().Unix()),
		Services:        uint64(s.p2p.Config().Services),
		GraphState:      s.getGraphState(),
		UserAgent:       []byte(s.p2p.Config().UserAgent),
		DisableRelayTx:  s.p2p.Config().DisableRelayTx,
	}

	return cs
}

func (s *Sync) getGraphState() *pb.GraphState {
	bs := s.p2p.BlockChain().BestSnapshot()

	gs := &pb.GraphState{
		Total:      uint32(bs.GraphState.GetTotal()),
		Layer:      uint32(bs.GraphState.GetLayer()),
		MainHeight: uint32(bs.GraphState.GetMainHeight()),
		MainOrder:  uint32(bs.GraphState.GetMainOrder()),
		Tips:       []*pb.Hash{},
	}
	count := 0
	for _, tip := range bs.GraphState.GetTipsList() {
		gs.Tips = append(gs.Tips, &pb.Hash{Hash: tip.Bytes()})
		count++
		if count >= MaxPBGraphStateTips {
			break
		}
	}

	return gs
}
