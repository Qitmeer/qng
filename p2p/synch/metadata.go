/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

// metaDataHandler reads the incoming metadata rpc request from the peer.
func (s *Sync) metaDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	return s.EncodeResponseMsg(stream, s.p2p.Metadata())
}

func (s *Sync) sendMetaDataRequest(stream network.Stream, pe *peers.Peer) (*pb.MetaData, *common.Error) {

	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("meta date request rsp")
		return nil, e
	}
	msg := new(pb.MetaData)
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}
