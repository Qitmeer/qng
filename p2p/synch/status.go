/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	"github.com/Qitmeer/qng/p2p/runutil"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

const (
	ReconnectionTime = time.Second * 30
)

// maintainPeerStatuses by infrequently polling peers for their latest status.
func (s *Sync) maintainPeerStatuses() {
	runutil.RunEvery(s.p2p.Context(), s.PeerInterval, func() {
		for _, pid := range s.Peers().Connected() {
			pe := s.peers.Get(pid)
			if pe == nil {
				continue
			}
			go func(id peer.ID) {
				// If our peer status has not been updated correctly we disconnect over here
				// and set the connection state over here instead.
				if s.p2p.Host().Network().Connectedness(id) != network.Connected {
					s.p2p.ConnectToPeer(pe.QAddress().String(), false)
					return
				}

				if !pe.CanConnectWithNetwork() {
					if err := s.sendGoodByeAndDisconnect(common.ErrBadPeer, pe); err != nil {
						log.Debug(fmt.Sprintf("Error when disconnecting with bad peer: %v", err))
					}
					return
				}

				if !pe.IsConsensus() {
					return
				}
				// If the status hasn't been updated in the recent interval time.
				if roughtime.Now().After(pe.ChainStateLastUpdated().Add(s.PeerInterval)) {
					if err := s.reValidatePeer(pe); err != nil {
						log.Debug(fmt.Sprintf("Failed to revalidate peer (%v), peer:%s", err, id))
					}
				}

				if pe.QNR() == nil && time.Since(pe.ConnectionTime()) > ReconnectionTime && s.p2p.Node() != nil {
					s.peerSync.SyncQNR(pe, s.p2p.Node().String())
				}
			}(pid)
		}
		for _, pid := range s.Peers().Connecting() {
			pe := s.peers.Get(pid)
			if pe == nil {
				continue
			}
			go func(id peer.ID) {
				if s.p2p.Host().Network().Connectedness(id) != network.Connected {
					s.peerSync.ReConnect(pe)
					return
				}
				if roughtime.Now().After(pe.ChainStateLastUpdated().Add(s.PeerInterval)) {
					s.peerSync.ReConnect(pe)
					return
				}
			}(pid)
		}
		for _, pid := range s.Peers().Disconnected() {
			pe := s.peers.Get(pid)
			if pe == nil {
				continue
			}
			if s.p2p.Host().Network().Connectedness(pe.GetID()) == network.Connected {
				err := s.p2p.Disconnect(pe.GetID())
				if err != nil {
					log.Debug(err.Error())
				}
			}
			//
			node := pe.Node()
			if node == nil ||
				time.Since(pe.ConnectionTime()) < ReconnectionTime ||
				pe.IsBad() {
				continue
			}
			s.LookupNode(nil, node)
		}
	})
}

func (s *Sync) reValidatePeer(pe *peers.Peer) error {
	if _, err := s.Send(pe, RPCChainState, s.getChainState()); err != nil {
		return err
	}
	if !pe.IsConsensus() {
		return nil
	}
	// Do not return an error for ping requests.
	metadataSeq := s.p2p.MetadataSeq()
	if _, err := s.Send(pe, RPCPingTopic, &metadataSeq); err != nil {
		log.Debug(fmt.Sprintf("Could not ping peer:%v", err))
	}
	return nil
}
