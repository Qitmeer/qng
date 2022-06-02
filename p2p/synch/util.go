package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	libp2pcore "github.com/libp2p/go-libp2p-core"
	"time"
)

var (
	ErrPeerUnknown = common.NewError(common.ErrPeerUnknown, peers.ErrPeerUnknown)
)

func closeWriteSteam(stream libp2pcore.Stream) error {
	err := stream.CloseWrite()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to close write stream(%s %s %s):%v",stream.Conn().RemotePeer(),stream.Protocol(),stream.Stat().Direction,err))
	}
	return err
}

func resetSteam(stream libp2pcore.Stream) error {
	if stream == nil {
		return nil
	}
	err := stream.Reset()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to reset stream(%s %s %s):%v",stream.Conn().RemotePeer(),stream.Protocol(),stream.Stat().Direction,err))
	}
	return err
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
