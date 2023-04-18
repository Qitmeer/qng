/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"sync/atomic"
)

func (s *Sync) sendGraphStateRequest(ctx context.Context, pe *peers.Peer, gs *pb.GraphState) (*pb.GraphState, error) {
	ctx, cancel := context.WithTimeout(ctx, ReqTimeout)
	defer cancel()

	stream, err := s.Send(ctx, gs, RPCGraphState, pe.GetID())
	if err != nil {
		return nil, err
	}

	code, errMsg, err := ReadRspCode(stream, s.p2p)
	if err != nil {
		return nil, err
	}

	if !code.IsSuccess() {
		s.Peers().IncrementBadResponses(stream.Conn().RemotePeer(), "graph state request rsp")
		closeStream(stream, s.p2p)
		return nil, errors.New(errMsg)
	}

	msg := &pb.GraphState{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, err
	}
	closeStream(stream, s.p2p)
	return msg, err
}

func (s *Sync) graphStateHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) *common.Error {
	pe := s.peers.Get(stream.Conn().RemotePeer())
	if pe == nil {
		return ErrPeerUnknown
	}

	ctx, cancel := context.WithTimeout(ctx, HandleTimeout)
	var err error
	defer func() {
		cancel()
	}()

	m, ok := msg.(*pb.GraphState)
	if !ok {
		err = fmt.Errorf("message is not type *pb.GraphState")
		return ErrMessage(err)
	}
	pe.UpdateGraphState(m)
	go s.peerSync.PeerUpdate(pe, false)

	e := s.EncodeResponseMsg(stream, s.getGraphState())
	if e != nil {
		return e
	}
	return nil
}

func (ps *PeerSync) processUpdateGraphState(pe *peers.Peer) error {
	if !pe.IsConnected() {
		err := fmt.Errorf("peer is not active")
		log.Warn(err.Error())
		return err
	}
	log.Trace(fmt.Sprintf("UpdateGraphState recevied from %v, state=%v ", pe.GetID(), pe.GraphState()))

	gs, err := ps.sy.sendGraphStateRequest(ps.sy.p2p.Context(), pe, ps.sy.getGraphState())
	if err != nil {
		log.Debug(err.Error())
		return err
	}
	pe.UpdateGraphState(gs)
	go ps.PeerUpdate(pe, false)
	return nil
}

func (ps *PeerSync) UpdateGraphState(pe *peers.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&ps.shutdown) != 0 {
		return
	}
	pe.RunRate(UpdateGraphState, UpdateGraphStateTime, func() {
		ps.processUpdateGraphState(pe)
	})
}
