/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	libp2pcore "github.com/libp2p/go-libp2p/core"
)

func (s *Sync) sendGoodByeMessage(message interface{}, pe *peers.Peer) *common.Error {
	codeV := message.(*uint64)
	code := common.ErrorCode(*codeV)
	logReason := fmt.Sprintf("Reason:%s", code.String())
	log.Debug(fmt.Sprintf("Sending Goodbye message to peer:%s (%s)", pe.IDWithAddress(), logReason))
	return nil
}

// goodbyeRPCHandler reads the incoming goodbye rpc message from the peer.
func (s *Sync) goodbyeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*uint64)
	if !ok {
		return ErrMessage(fmt.Errorf("wrong message type for goodbye, got %T, wanted *uint64", msg))
	}
	logReason := fmt.Sprintf("Reason:%s", common.ErrorCode(*m).String())
	log.Debug(fmt.Sprintf("Peer receive a goodbye message:%s (%s)", pe.IDWithAddress(), logReason))
	// closes all streams with the peer
	go s.peerSync.immediatelyDisconnected(pe)
	return nil
}

func (s *Sync) sendGoodByeAndDisconnect(code common.ErrorCode, pe *peers.Peer) error {
	codeV := uint64(code)
	_, err := s.Send(pe, RPCGoodByeTopic, &codeV)
	if err != nil {
		log.Debug(fmt.Sprintf("Could not send goodbye message: %v ", err))
	}
	go s.peerSync.immediatelyDisconnected(pe)
	return err
}
