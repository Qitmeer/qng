/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"bytes"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/encoder"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/libp2p/go-libp2p/core/network"
)

func generateErrorResponse(e *common.Error, encoding encoder.NetworkEncoding) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{byte(e.Code)})
	resp := &pb.ErrorResponse{
		Message: []byte(e.Code.String()),
	}
	if _, err := encoding.EncodeWithMaxLength(buf, resp); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ReadRspCode response from a RPC stream.
func ReadRspCode(stream network.Stream, rpc peers.P2PRPC) *common.Error {
	b := make([]byte, 1)
	_, err := stream.Read(b)
	if err != nil {
		return common.NewError(common.ErrStreamRead, err)
	}

	if b[0] == byte(common.ErrNone) {
		return common.NewSuccess()
	}

	if b[0] == byte(common.ErrDAGConsensus) {
		return common.NewError(common.ErrDAGConsensus, nil)
	}

	msg := &pb.ErrorResponse{
		Message: []byte{},
	}

	err = DecodeMessage(stream, rpc, msg)
	if err != nil {
		return common.NewError(common.ErrStreamRead, err)
	}
	return common.NewErrorStr(common.ErrorCode(b[0]), string(msg.Message))
}
