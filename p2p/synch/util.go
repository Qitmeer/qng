package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/v2/p2p/common"
	"github.com/Qitmeer/qng/v2/p2p/peers"
	"github.com/ethereum/go-ethereum/p2p/enode"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"strconv"
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
	UpdateGraphStateTime = time.Second / 2
	DefaultRateTaskTime  = time.Second * 2
)

const (
	UpdateGraphState = "UpdateGraphState"
	PeerUpdate       = "PeerUpdate"
	PeerUpdateOrphan = "PeerUpdateOrphan"
)

func CreateMeerNode(addr ma.Multiaddr, url string) (*enode.Node, error) {
	en := enode.MustParse(url)
	ip, err := manet.ToIP(addr)
	if err != nil {
		return nil, err
	}
	tcpstr, err := addr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, err
	}
	tcp, err := strconv.Atoi(tcpstr)
	if err != nil {
		return nil, err
	}
	return enode.NewV4(en.Pubkey(), ip, tcp, 0), nil
}
