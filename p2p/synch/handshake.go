/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"bufio"
	"fmt"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/p2p/peers"
	"github.com/multiformats/go-multistream"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// The time to wait for a chain state request.
	timeForChainState = 10 * time.Second

	timeForBidirChan = 4 * time.Second

	timeForBidirChanLife = 10 * time.Minute
)

func (ps *PeerSync) Connected(pid peer.ID, conn network.Conn) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&ps.shutdown) != 0 {
		return
	}

	//ps.msgChan <- &ConnectedMsg{ID: pid, Conn: conn}
	go ps.processConnected(&ConnectedMsg{ID: pid, Conn: conn})
}

func (ps *PeerSync) processConnected(msg *ConnectedMsg) {
	pe := ps.sy.peers.Fetch(msg.ID)
	pe.HSlock.Lock()
	defer pe.HSlock.Unlock()

	conn := msg.Conn

	pe.UpdateAddrDir(nil, conn.RemoteMultiaddr(), conn.Stat().Direction)
	pe.IncreaseReConnect()
	// Handle the various pre-existing conditions that will result in us not handshaking.
	if pe.IsConnected() {
		log.Trace(fmt.Sprintf("%s currentState:%d reason:already connected, Ignoring connection request", pe.IDWithAddress(), pe.ConnectionState()))
		return
	}

	// Do not perform handshake on inbound dials.
	if conn.Stat().Direction == network.DirInbound {
		return
	}

	if err := ps.sy.reValidatePeer(pe); err != nil {
		log.Trace(fmt.Sprintf("%s Handshake failed (%s)", pe.IDWithAddress(), err))
		return
	}
	ps.Connection(pe)
}

func (ps *PeerSync) immediatelyConnected(pe *peers.Peer) {
	pe.HSlock.Lock()
	defer pe.HSlock.Unlock()

	ps.Connection(pe)
}

func (ps *PeerSync) Connection(pe *peers.Peer) {
	if pe.ConnectionState().IsConnected() {
		return
	}
	pe.SetConnectionState(peers.PeerConnected)
	// Go through the handshake process.
	multiAddr := fmt.Sprintf("%s/p2p/%s", pe.Address().String(), pe.GetID().String())

	if !pe.IsConsensus() {
		log.Info(fmt.Sprintf("%s direction:%s multiAddr:%s  (%s)",
			pe.GetID(), pe.Direction(), multiAddr, pe.Services().String()))
		return
	}
	log.Info(fmt.Sprintf("%s direction:%s multiAddr:%s activePeers:%d Peer Connected",
		pe.GetID(), pe.Direction(), multiAddr, len(ps.sy.peers.Active())))

	go ps.OnPeerConnected(pe)
}

func (ps *PeerSync) immediatelyDisconnected(pe *peers.Peer) {
	pe.HSlock.Lock()
	defer pe.HSlock.Unlock()

	ps.Disconnect(pe)
}

func (ps *PeerSync) Disconnect(pe *peers.Peer) {
	if !pe.IsConnected() {
		return
	}
	pe.SetConnectionState(peers.PeerDisconnected)
	if !pe.IsConsensus() {
		if pe.Services() == protocol.Unknown {
			log.Trace(fmt.Sprintf("Disconnect:%v ", pe.IDWithAddress()))
		} else {
			log.Trace(fmt.Sprintf("Disconnect:%v (%s)", pe.IDWithAddress(), pe.Services().String()))
		}
		return
	}

	log.Trace(fmt.Sprintf("Disconnect:%v ", pe.IDWithAddress()))
	go ps.OnPeerDisconnected(pe)
}

func (ps *PeerSync) ReConnect(pe *peers.Peer) error {
	pe.HSlock.Lock()
	ps.Disconnect(pe)
	pe.HSlock.Unlock()

	return ps.sy.p2p.ConnectToPeer(pe.QAddress().String(), false)
}

func (ps *PeerSync) TryDisconnect(pe *peers.Peer) {
	pe.HSlock.Lock()
	ps.Disconnect(pe)
	pe.HSlock.Unlock()

	if err := ps.sy.p2p.Disconnect(pe.GetID()); err != nil {
		log.Error(fmt.Sprintf("%s Unable to disconnect from peer:%v", pe.GetID(), err))
	}
}

// AddConnectionHandler adds a callback function which handles the connection with a
// newly added peer. It performs a handshake with that peer by sending a hello request
// and validating the response from the peer.
func (s *Sync) AddConnectionHandler() {
	s.connectionNotify = &network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			remotePeer := conn.RemotePeer()
			log.Trace(fmt.Sprintf("ConnectedF:%s, %v ", remotePeer, conn.RemoteMultiaddr()))
			s.peerSync.Connected(remotePeer, conn)
		},
	}
	s.p2p.Host().Network().Notify(s.connectionNotify)
}

func (ps *PeerSync) Disconnected(pid peer.ID, conn network.Conn) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&ps.shutdown) != 0 {
		return
	}

	//ps.msgChan <- &DisconnectedMsg{ID: pid, Conn: conn}
	//go ps.processDisconnected(&DisconnectedMsg{ID: pid, Conn: conn})
}

func (ps *PeerSync) processDisconnected(msg *DisconnectedMsg) {
	// Must be handled in a goroutine as this callback cannot be blocking.
	pe := ps.sy.peers.Get(msg.ID)
	if pe == nil {
		return
	}

	pe.HSlock.Lock()
	defer pe.HSlock.Unlock()
	ps.Disconnect(pe)
}

// AddDisconnectionHandler disconnects from peers.  It handles updating the peer status.
// This also calls the handler responsible for maintaining other parts of the sync or p2p system.
func (s *Sync) AddDisconnectionHandler() {
	s.disconnectionNotify = &network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			remotePeer := conn.RemotePeer()
			log.Trace(fmt.Sprintf("DisconnectedF:%s", remotePeer))
			s.peerSync.Disconnected(remotePeer, conn)
		},
	}
	s.p2p.Host().Network().Notify(s.disconnectionNotify)
}

func (s *Sync) bidirectionalChannelCapacity(pe *peers.Peer, conn network.Conn) bool {
	if conn.Stat().Direction == network.DirOutbound {
		pe.SetBidChanCap(time.Now())
		return true
	}
	if s.p2p.Config().IsCircuit || s.IsWhitePeer(pe.GetID()) {
		pe.SetBidChanCap(time.Now())
		return true
	}

	bidChanLife := pe.GetBidChanCap()
	if !bidChanLife.IsZero() {
		if time.Since(bidChanLife) < timeForBidirChanLife {
			return true
		}
	}

	//
	peAddr := conn.RemoteMultiaddr()
	ipAddr := ""
	protocol := ""
	port := ""
	ps := peAddr.Protocols()
	if len(ps) >= 1 {
		ia, err := peAddr.ValueForProtocol(ps[0].Code)
		if err != nil {
			log.Debug(err.Error())
			pe.SetBidChanCap(time.Time{})
			return false
		}
		ipAddr = ia
	}
	if len(ps) >= 2 {
		protocol = ps[1].Name
		po, err := peAddr.ValueForProtocol(ps[1].Code)
		if err != nil {
			log.Debug(err.Error())
			pe.SetBidChanCap(time.Time{})
			return false
		}
		port = po
	}
	if len(ipAddr) <= 0 ||
		len(protocol) <= 0 ||
		len(port) <= 0 {
	}
	bidConn, err := net.DialTimeout(protocol, fmt.Sprintf("%s:%s", ipAddr, port), timeForBidirChan)
	if err != nil {
		log.Debug(err.Error())
		pe.SetBidChanCap(time.Time{})
		return false
	}
	reply, err := bufio.NewReader(bidConn).ReadString('\n')
	if err != nil {
		log.Debug(err.Error())
		pe.SetBidChanCap(time.Time{})
		return false
	}
	if !strings.Contains(reply, multistream.ProtocolID) {
		log.Debug(fmt.Sprintf("BidChan protocol is error"))
		pe.SetBidChanCap(time.Time{})
		return false
	}
	log.Debug(fmt.Sprintf("Bidirectional channel capacity:%s", pe.GetID().String()))
	bidConn.Write([]byte(fmt.Sprintf("%s\n", multistream.ProtocolID)))
	bidConn.Close()

	pe.SetBidChanCap(time.Now())
	return true
}

func (s *Sync) IsWhitePeer(pid peer.ID) bool {
	_, ok := s.LANPeers[pid]
	return ok
}

func (s *Sync) IsPeerAtLimit() bool {
	//numOfConns := len(s.p2p.Host().Network().Peers())
	maxPeers := int(s.p2p.Config().MaxPeers)
	activePeers := len(s.Peers().Active())

	return activePeers >= maxPeers
}

func (s *Sync) IsInboundPeerAtLimit() bool {
	return len(s.Peers().DirInbound()) >= s.p2p.Config().MaxInbound
}

func (s *Sync) ConnectionGater(pid *peer.ID, dir network.Direction) bool {
	if pid != nil {
		pe := s.peers.Get(*pid)
		if pe != nil {
			delay := time.Since(pe.ChainStateLastUpdated())
			if delay <= time.Hour*24 && !pe.CanConnectWithNetwork() {
				return false
			}
		}

		// generic
		if s.IsWhitePeer(*pid) {
			return true
		}
	}

	if s.IsPeerAtLimit() {
		if pid != nil {
			log.Trace(fmt.Sprintf("connectionGater  peer:%s reason:at peer max limit", pid.String()))
		} else {
			log.Trace("connectionGater reason:at peer max limit")
		}

		return false
	}
	if dir == network.DirInbound {
		if s.IsInboundPeerAtLimit() {
			if pid != nil {
				log.Trace(fmt.Sprintf("peer:%s reason:at peer limit,Not accepting inbound dial", pid.String()))
			} else {
				log.Trace("reason:at peer limit,Not accepting inbound dial")
			}

			return false
		}
	}
	return true
}
