package synch

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"time"
)

func (ps *PeerSync) establishMeerConnection(pe *peers.Peer) {
	if !ps.checkMeerConnection(pe) {
		return
	}
	err := ps.sy.establishMeerConnection(pe)
	if err != nil {
		log.Error(err.Error())
	}
}

func (ps *PeerSync) checkMeerConnection(pe *peers.Peer) bool {
	if !ps.IsRunning() {
		return false
	}
	if pe.GetMeerConn() {
		return false
	}
	if !pe.IsSupportMeerP2PBridging() {
		return false
	}
	if pe.Direction() != network.DirOutbound {
		return false
	}
	ms := pe.GetMeerState()
	if ms == nil {
		return false
	}
	return true
}

func (s *Sync) establishMeerConnection(pe *peers.Peer) error {
	proto := RPCMeerConn
	if !s.peerSync.IsRunning() {
		return errors.New("No run PeerSync")
	}
	if pe.Direction() != network.DirOutbound {
		return fmt.Errorf("Peer (%s) is not outbound", pe.GetID().String())
	}
	ms := pe.GetMeerState()
	if ms == nil {
		return fmt.Errorf("Peer (%s) no meer state", pe.GetID().String())
	}
	url := ms.Enode
	if len(url) <= 0 {
		url = ms.ENR
	}
	if len(url) <= 0 {
		return fmt.Errorf("Peer (%s) no meer state url", pe.GetID().String())
	}
	dest, err := CreateMeerNode(pe.Address(), url)
	if err != nil {
		return err
	}

	log.Trace("Send message", "protocol", getProtocol(s.p2p, proto), "peer", pe.IDWithAddress())

	curState := s.p2p.Host().Network().Connectedness(pe.GetID())
	if curState == network.CannotConnect {
		return common.NewErrorStr(common.ErrLibp2pConnect, curState.String()).ToError()
	}

	topic := getProtocol(s.p2p, proto)

	stream, err := s.p2p.Host().NewStream(context.Background(), pe.GetID(), protocol.ID(topic))
	if err != nil {
		return common.NewErrorStr(common.ErrStreamBase, fmt.Sprintf("open stream on topic %v failed", topic)).ToError()
	}

	common.EgressConnectMeter.Mark(1)
	qc, err := NewMeerConn(stream)
	if err != nil {
		return err
	}
	pe.SetMeerConn(true)

	go func() {
		_, err := s.p2p.MeerServer().Connect(qc, dest)
		if err != nil {
			log.Error(err.Error())
		}
		pe.SetMeerConn(false)
		err = stream.Close()
		if err != nil {
			log.Error("Close meer conn", "err", err.Error())
		}
	}()
	return err
}

func (s *Sync) registerMeerConnection() {
	basetopic := RPCMeerConn
	topic := getProtocol(s.p2p, basetopic)

	s.p2p.Host().SetStreamHandler(protocol.ID(topic), func(stream network.Stream) {
		if !s.p2p.IsRunning() {
			log.Debug("PeerSync is not running, ignore the handling", "protocol", topic)
			stream.Close()
			return
		}
		common.IngressConnectMeter.Mark(1)
		pe := s.p2p.Peers().Get(stream.Conn().RemotePeer())
		if (pe == nil || pe.ChainState() == nil) && basetopic != RPCChainState && basetopic != RPCChainStateV2 && basetopic != RPCGoodByeTopic {
			log.Debug("Peer is not init, ignore the handling", "protocol", topic, "pe", stream.Conn().RemotePeer())
			stream.Close()
			return
		}

		if pe == nil {
			stream.Close()
			return
		}
		pe.UpdateAddrDir(nil, stream.Conn().RemoteMultiaddr(), stream.Conn().Stat().Direction)

		log.Trace("Stream handler", "protocol", topic, "peer", pe.IDWithAddress())
		qc, err := NewMeerConn(stream)
		if err != nil {
			log.Error(err.Error())
			stream.Close()
			return
		}
		pe.SetMeerConn(true)
		defer pe.SetMeerConn(false)

		_, err = s.p2p.MeerServer().Connect(qc, nil)
		if err != nil {
			log.Error(err.Error())
			stream.Close()
			return
		}
	})
}

func (ps *PeerSync) meerSync(target chan ecommon.Hash, quit chan struct{}) {
	log.Info("enter meer sync")
	defer log.Info("exit meer sync")

	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	var tar ecommon.Hash

	for {
		if !ps.IsRunning() {
			log.Warn("Quit meerevm sync")
			return
		}
		if ps.snapStatus.IsEVMCompleted() {
			return
		}
		select {
		case tar = <-target:
			if tar == (ecommon.Hash{}) {
				continue
			}
			err := ps.sy.p2p.BlockChain().MeerChain().Downloader().SyncQngWaitPeers(tar, ps.quit, time.Minute*5)
			if err != nil {
				log.Info("Failed to trigger beacon sync", "err", err)
				continue
			}
		case <-ticker.C:
			if tar == (ecommon.Hash{}) {
				continue
			}
			block := ps.sy.p2p.BlockChain().MeerChain().Ether().BlockChain().GetBlockByHash(tar)
			if block != nil {
				ps.snapStatus.CompleteEVM()
				log.Info("Sync target reached", "number", block.NumberU64(), "hash", block.Hash())
				return
			}
		case <-ps.quit:
			log.Warn("Quit meerevm sync by ps")
			return
		case <-quit:
			log.Warn("Quit meerevm sync")
			return
		}

	}

}
