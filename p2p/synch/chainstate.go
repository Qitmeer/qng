/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
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

func (s *Sync) chainStateHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.ChainState)
	if !ok {
		return ErrMessage(fmt.Errorf("message is not type *pb.ChainState"))
	}
	e := s.validateChainStateMessage(m, pe)
	if e != nil {
		if e.Code.IsDAGConsensus() {
			// Respond with our status and disconnect with the peer.
			s.UpdateChainState(pe, m, false)
			if err := s.EncodeResponseMsgPro(stream, s.getChainState(), e.Code); err != nil {
				return err
			}
		}
		return e
	}
	if !s.bidirectionalChannelCapacity(pe, stream.Conn()) {
		s.UpdateChainState(pe, m, false)
		if err := s.EncodeResponseMsgPro(stream, s.getChainState(), common.ErrDAGConsensus); err != nil {
			return err
		}
		return ErrMessage(fmt.Errorf("bidirectional channel capacity"))
	}
	s.UpdateChainState(pe, m, true)
	return s.EncodeResponseMsg(stream, s.getChainState())
}

func (s *Sync) UpdateChainState(pe *peers.Peer, chainState *pb.ChainState, action bool) {
	pe.SetChainState(chainState)
	if !action {
		go s.peerSync.immediatelyDisconnected(pe)
		return
	}
	go s.peerSync.immediatelyConnected(pe)
}

func (s *Sync) validateChainStateMessage(msg *pb.ChainState, pe *peers.Peer) *common.Error {
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
	// state root check
	gs := changePBGraphStateToGraphState(msg.GraphState)
	if gs != nil {
		bs := s.p2p.BlockChain().BestSnapshot()
		pmt := gs.GetMainChainTip()
		if bs.GraphState != nil &&
			pmt != nil &&
			bs.GraphState.GetMainChainTip().IsEqual(pmt) {
			sr := changePBHashToHash(msg.StateRoot)
			if !bs.StateRoot.IsEqual(sr) {
				return common.NewError(common.ErrDAGConsensus,
					fmt.Errorf("State root inconsistent:me(%s) != peer(%s) in block %s order(%d)", bs.StateRoot.String(), sr.String(), bs.Hash.String(), bs.GraphState.GetMainOrder()))
			}
		}
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
		StateRoot:       &pb.Hash{Hash: s.p2p.BlockChain().BestSnapshot().StateRoot.Bytes()},
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
