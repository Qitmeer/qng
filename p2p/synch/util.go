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

func closeWriteStream(stream libp2pcore.Stream) error {
	err := stream.CloseWrite()
	if err != nil {
		return fmt.Errorf("Failed to close write stream:%v", err)
	}
	return err
}

func closeStream(stream libp2pcore.Stream) error {
	err := stream.Close()
	if err != nil {
		return fmt.Errorf("Failed to close stream:%v", err)
	}
	return err
}

func resetStream(stream libp2pcore.Stream) error {
	if stream == nil {
		return nil
	}
	err := stream.Reset()
	if err != nil {
		return fmt.Errorf("Failed to reset stream(%s %s):%v", stream.Protocol(), stream.Stat().Direction, err)
	}
	return err
}

func DecodeMessage(stream libp2pcore.Stream, rpc peers.P2PRPC, msg interface{}) error {
	err := rpc.Encoding().DecodeWithMaxLength(stream, msg)
	if err != nil {
		return err
	}
	return nil
}

func EncodeMessage(stream libp2pcore.Stream, rpc peers.P2PRPC, msg interface{}) (int, error) {
	size, err := rpc.Encoding().EncodeWithMaxLength(stream, msg)
	if err != nil {
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
