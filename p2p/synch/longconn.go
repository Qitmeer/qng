package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/v2/p2p/common"
	"github.com/Qitmeer/qng/v2/p2p/peers"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func (ps *PeerSync) establishLongConnection(pe *peers.Peer) {
	if !ps.checkLongConnection(pe) {
		return
	}
	err := ps.sy.establishLongConnection(pe)
	if err != nil {
		log.Error(err.Error())
	}
}

func (ps *PeerSync) checkLongConnection(pe *peers.Peer) bool {
	if !ps.IsRunning() {
		return false
	}
	if !pe.IsSupportLongConn() {
		return false
	}
	if pe.IsLongConn() {
		return false
	}
	if pe.Direction() != network.DirOutbound {
		return false
	}
	return true
}

func (s *Sync) establishLongConnection(pe *peers.Peer) error {
	proto := RPCLongConn
	log.Trace("Send message", "protocol", getProtocol(s.p2p, proto), "peer", pe.IDWithAddress())

	curState := s.p2p.Host().Network().Connectedness(pe.GetID())
	if curState == network.CannotConnect || curState == network.NotConnected {
		return common.NewErrorStr(common.ErrLibp2pConnect, curState.String()).ToError()
	}

	topic := getProtocol(s.p2p, proto)
	ctx := context.Background()
	stream, err := s.p2p.Host().NewStream(ctx, pe.GetID(), protocol.ID(topic))
	if err != nil {
		return common.NewErrorStr(common.ErrStreamBase, fmt.Sprintf("open stream on topic %v failed", topic)).ToError()
	}
	common.EgressConnectMeter.Mark(1)

	go func() {
		err = pe.Run(stream, s.p2p.Encoding())
		if err != nil {
			log.Warn(err.Error())
		}
		err = stream.Close()
		if err != nil {
			log.Warn(err.Error())
		}
	}()
	return nil
}

func (s *Sync) registerLongConnection() {
	basetopic := RPCLongConn
	topic := getProtocol(s.p2p, basetopic)

	s.p2p.Host().SetStreamHandler(protocol.ID(topic), func(stream network.Stream) {
		defer func() {
			err := stream.Close()
			if err != nil {
				log.Error(err.Error())
			}
		}()
		if !s.p2p.IsRunning() {
			log.Debug("PeerSync is not running, ignore the handling", "protocol", topic)
			return
		}
		common.IngressConnectMeter.Mark(1)
		pe := s.p2p.Peers().Get(stream.Conn().RemotePeer())
		if (pe == nil || pe.ChainState() == nil) && basetopic != RPCChainState && basetopic != RPCChainStateV2 && basetopic != RPCGoodByeTopic {
			log.Debug("Peer is not init, ignore the handling", "protocol", topic, "pe", stream.Conn().RemotePeer())
			return
		}

		if pe == nil {
			return
		}
		pe.UpdateAddrDir(nil, stream.Conn().RemoteMultiaddr(), stream.Conn().Stat().Direction)

		log.Trace("Stream handler", "protocol", topic, "peer", pe.IDWithAddress())
		if pe.IsLongConn() {
			log.Warn("Stream handler duplicate", "protocol", topic, "peer", pe.IDWithAddress())
			return
		}
		err := pe.Run(stream, s.p2p.Encoding())
		if err != nil {
			log.Warn(err.Error())
		}
	})
}
