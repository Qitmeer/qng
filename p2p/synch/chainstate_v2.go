/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/common/roughtime"
	"github.com/Qitmeer/qng/v2/core/protocol"
	"github.com/Qitmeer/qng/v2/p2p/common"
	"github.com/Qitmeer/qng/v2/p2p/peers"
	pb "github.com/Qitmeer/qng/v2/p2p/proto/v1"
	v2 "github.com/Qitmeer/qng/v2/p2p/proto/v2"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

func (s *Sync) sendChainStateV2Request(stream network.Stream, pe *peers.Peer) *common.Error {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() && !e.Code.IsDAGConsensus() {
		e.Add("chain state request")
		return e
	}
	msg := &v2.ChainState{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return common.NewError(common.ErrStreamRead, err)
	}
	s.UpdateChainState(pe, msg, !e.Code.IsDAGConsensus())
	if !e.Code.IsSuccess() {
		return e
	}
	e = s.validateChainStateMessage(msg, pe)
	if e != nil {
		if e.Code.IsDAGConsensus() {
			go s.sendGoodByeAndDisconnect(common.ErrDAGConsensus, pe)
		}
		return e
	}
	return e
}

func (s *Sync) chainStateV2Handler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*v2.ChainState)
	if !ok {
		return ErrMessage(fmt.Errorf("message is not type *pb.ChainState"))
	}
	e := s.validateChainStateMessage(m, pe)
	if e != nil {
		if e.Code.IsDAGConsensus() {
			// Respond with our status and disconnect with the peer.
			s.UpdateChainState(pe, m, false)
			if err := s.EncodeResponseMsgPro(stream, s.getChainStateV2(), e.Code); err != nil {
				return err
			}
		}
		return e
	}
	if !s.bidirectionalChannelCapacity(pe, stream.Conn()) {
		s.UpdateChainState(pe, m, false)
		if err := s.EncodeResponseMsgPro(stream, s.getChainStateV2(), common.ErrDAGConsensus); err != nil {
			return err
		}
		return ErrMessage(fmt.Errorf("bidirectional channel capacity"))
	}
	s.UpdateChainState(pe, m, true)
	return s.EncodeResponseMsg(stream, s.getChainStateV2())
}

func (s *Sync) UpdateChainState(pe *peers.Peer, chainState *v2.ChainState, action bool) {
	pe.SetChainState(chainState)
	if !action {
		go s.peerSync.immediatelyDisconnected(pe)
		return
	}
	go s.peerSync.immediatelyConnected(pe)
}

func (s *Sync) validateChainStateMessage(msg *v2.ChainState, pe *peers.Peer) *common.Error {
	if msg == nil {
		return common.NewErrorStr(common.ErrGeneric, "msg is nil")
	}
	if protocol.HasServices(protocol.ServiceFlag(msg.Services), protocol.Relay) {
		return nil
	}
	if protocol.HasServices(protocol.ServiceFlag(msg.Services), protocol.Observer) {
		return nil
	}
	genesisHash := s.p2p.GetGenesisHash()
	msgGenesisHash, err := hash.NewHash(msg.GenesisHash.Hash)
	if err != nil {
		return common.NewErrorStr(common.ErrGeneric, "invalid genesis")
	}
	if !msgGenesisHash.IsEqual(genesisHash) {
		return common.NewErrorStr(common.ErrDAGConsensus, "invalid genesis")
	}
	// Notify and disconnect clients that have a protocol version that is
	// too old.
	if msg.ProtocolVersion < uint32(protocol.InitialProcotolVersion) {
		return common.NewError(common.ErrDAGConsensus, fmt.Errorf("protocol version must be %d or greater",
			protocol.InitialProcotolVersion))
	}
	if msg.GraphState.Total <= 0 {
		return common.NewErrorStr(common.ErrDAGConsensus, "invalid graph state")
	}
	if pe.Direction() == network.DirInbound {
		// Reject outbound peers that are not full nodes.
		wantServices := protocol.Full
		if !protocol.HasServices(protocol.ServiceFlag(msg.Services), wantServices) {
			// missingServices := wantServices & ^msg.Services
			missingServices := protocol.MissingServices(protocol.ServiceFlag(msg.Services), wantServices)
			return common.NewErrorStr(common.ErrDAGConsensus, fmt.Sprintf("Rejecting peer %s with services %v "+
				"due to not providing desired services %v\n", pe.GetID().String(), msg.Services, missingServices))
		}
	}

	return nil
}

func (s *Sync) getChainStateV2() *v2.ChainState {
	genesisHash := s.p2p.GetGenesisHash()

	cs := &v2.ChainState{
		GenesisHash:     &pb.Hash{Hash: genesisHash.Bytes()},
		ProtocolVersion: s.p2p.Config().ProtocolVersion,
		Timestamp:       uint64(roughtime.Now().Unix()),
		Services:        uint64(s.p2p.Config().Services),
		GraphState:      s.getGraphState(),
		UserAgent:       []byte(s.p2p.Config().UserAgent),
		DisableRelayTx:  s.p2p.Config().DisableRelayTx,
		MeerState:       s.getMeerState(),
		SnapSync:        s.peerSync.IsSnapSync(),
	}

	return cs
}

func (s *Sync) getMeerState() *v2.MeerState {
	mc := s.p2p.BlockChain().MeerChain()

	ms := &v2.MeerState{
		Id:     &pb.Hash{Hash: mc.Server().LocalNode().ID().Bytes()},
		Number: mc.CurrentHeader().Number.Uint64(),
		Enode:  []byte(mc.Server().LocalNode().Node().URLv4()),
		Enr:    []byte(mc.Server().LocalNode().Node().String()),
	}

	return ms
}
