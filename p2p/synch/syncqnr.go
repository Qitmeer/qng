/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/qnode"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"sync/atomic"
)

func (s *Sync) sendQNRRequest(stream network.Stream, pe *peers.Peer) (*pb.SyncQNR, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("QNR request rsp")
		return nil, e
	}
	msg := &pb.SyncQNR{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}

func (s *Sync) QNRHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.SyncQNR)
	if !ok {
		err := fmt.Errorf("message is not type *pb.GraphState")
		return ErrMessage(err)
	}

	if pe.QNR() == nil {
		err := s.peerSync.LookupNode(pe, string(m.Qnr))
		if err != nil {
			return ErrMessage(err)
		}
	}

	if s.p2p.Node() == nil {
		return ErrMessage(fmt.Errorf("Disable Node V5"))
	}
	return s.EncodeResponseMsg(stream, &pb.SyncQNR{Qnr: []byte(s.p2p.Node().String())})
}

func (s *Sync) LookupNode(pe *peers.Peer, peNode *qnode.Node) {
	pnResult := s.p2p.Resolve(peNode)
	if pnResult != nil {
		if pe != nil {
			pe.SetQNR(pnResult.Record())
		}
		log.Debug(fmt.Sprintf("Lookup success: %s", pnResult.ID()))
	} else {
		log.Debug(fmt.Sprintf("Lookup fail: %s", peNode.ID()))
	}
}

func (ps *PeerSync) processQNR(msg *SyncQNRMsg) error {
	if !msg.pe.IsConnected() {
		return fmt.Errorf("peer is not active")
	}
	ret, err := ps.sy.Send(msg.pe, RPCSyncQNR, &pb.SyncQNR{Qnr: []byte(msg.qnr)})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	qnr := ret.(*pb.SyncQNR)
	if msg.pe.QNR() == nil {
		return ps.LookupNode(msg.pe, string(qnr.Qnr))
	}
	return nil
}

func (ps *PeerSync) SyncQNR(pe *peers.Peer, qnr string) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&ps.shutdown) != 0 {
		return
	}

	ps.msgChan <- &SyncQNRMsg{pe: pe, qnr: qnr}
}

func (ps *PeerSync) LookupNode(pe *peers.Peer, qnr string) error {
	peerNode, err := qnode.Parse(qnode.ValidSchemes, qnr)
	if err != nil {
		return err
	}
	ps.sy.LookupNode(pe, peerNode)
	return nil
}
