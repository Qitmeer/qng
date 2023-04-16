/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
)

// metaDataHandler reads the incoming metadata rpc request from the peer.
func (s *Sync) metaDataHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) *common.Error {
	ctx, cancel := context.WithTimeout(ctx, HandleTimeout)
	defer cancel()
	return s.EncodeResponseMsg(stream, s.p2p.Metadata())
}

func (s *Sync) sendMetaDataRequest(ctx context.Context, id peer.ID) (*pb.MetaData, error) {
	ctx, cancel := context.WithTimeout(ctx, ReqTimeout)
	defer cancel()

	stream, err := s.Send(ctx, new(interface{}), RPCMetaDataTopic, id)
	if err != nil {
		return nil, err
	}
	// we close the stream outside of `send` because
	// metadata requests send no payload, so closing the
	// stream early leads it to a reset.
	code, errMsg, err := ReadRspCode(stream, s.p2p)
	if err != nil {
		return nil, err
	}
	if code != 0 {
		s.Peers().IncrementBadResponses(stream.Conn().RemotePeer(), "meta date request rsp")
		closeStream(stream, s.p2p)
		return nil, fmt.Errorf(errMsg)
	}
	msg := new(pb.MetaData)
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, err
	}
	closeStream(stream, s.p2p)
	return msg, nil
}
