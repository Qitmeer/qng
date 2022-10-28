package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"time"
)

var (
	ErrPeerUnknown = common.NewError(common.ErrPeerUnknown, peers.ErrPeerUnknown)
)

func closeWriteSteam(stream libp2pcore.Stream, rpc common.P2PRPC) error {
	err := stream.CloseWrite()
	if err != nil {
		log.Debug(fmt.Sprintf("Failed to close write stream(%s %s %s):%v", stream.Conn().RemotePeer(), stream.Protocol(), stream.Stat().Direction, err))
		processUnderlyingError(rpc, stream.Conn().RemotePeer(), err)
	}
	return err
}

func resetSteam(stream libp2pcore.Stream, rpc common.P2PRPC) error {
	if stream == nil {
		return nil
	}
	err := stream.Reset()
	if err != nil {
		log.Debug(fmt.Sprintf("Failed to reset stream(%s %s %s):%v", stream.Conn().RemotePeer(), stream.Protocol(), stream.Stat().Direction, err))
		processUnderlyingError(rpc, stream.Conn().RemotePeer(), err)
	}
	return err
}

func DecodeMessage(stream libp2pcore.Stream, rpc common.P2PRPC, msg interface{}) error {
	err := rpc.Encoding().DecodeWithMaxLength(stream, msg)
	if err != nil {
		processUnderlyingError(rpc, stream.Conn().RemotePeer(), err)
		return err
	}
	return nil
}

func EncodeMessage(stream libp2pcore.Stream, rpc common.P2PRPC, msg interface{}) (int, error) {
	size, err := rpc.Encoding().EncodeWithMaxLength(stream, msg)
	if err != nil {
		processUnderlyingError(rpc, stream.Conn().RemotePeer(), err)
		return size, err
	}
	return size, nil
}

func ErrMessage(err error) *common.Error {
	return common.NewError(common.ErrMessage, err)
}

func ErrDAGConsensus(err error) *common.Error {
	return common.NewError(common.ErrDAGConsensus, err)
}

const (
	UpdateGraphStateTime = time.Second * 2
	DefaultRateTaskTime  = time.Second * 2
)

const (
	UpdateGraphState = "UpdateGraphState"
	PeerUpdate       = "PeerUpdate"
	PeerUpdateOrphan = "PeerUpdateOrphan"
)
