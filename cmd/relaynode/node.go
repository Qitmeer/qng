/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package main

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/boot"
	rconfig "github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	pv "github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/p2p"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/encoder"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/synch"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc"
	ds "github.com/ipfs/go-ds-leveldb"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoreds"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"path"
	"reflect"
	"sync"
)

type Node struct {
	service.Service
	cfg *rconfig.Config

	privateKey crypto.PrivKey

	host host.Host

	peerStatus *peers.Status

	hslock sync.RWMutex
}

func (node *Node) init(cfg *rconfig.Config) error {
	log.Info(fmt.Sprintf("Start relay node..."))
	node.InitContext()
	node.InitServices()

	err := cfg.Load()
	if err != nil {
		return err
	}
	node.cfg = cfg

	pk, err := common.PrivateKey(cfg.DataDir, cfg.PrivateKey, 0600)
	if err != nil {
		return err
	}
	node.privateKey = pk

	//
	node.peerStatus = peers.NewStatus(node)

	if err := node.RegisterRpcService(); err != nil {
		return err
	}
	if err := node.RegisterBootService(); err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Load config completed"))
	log.Info(fmt.Sprintf("NetWork:%s  Genesis:%s", params.ActiveNetParams.Name, params.ActiveNetParams.GenesisHash.String()))
	return nil
}

func (node *Node) Stop() error {
	if err := node.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop relay node"))
	return nil
}

func (node *Node) Start() error {
	if err := node.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Run relay node..."))
	err := node.startP2P()
	if err != nil {
		return err
	}

	interrupt := system.InterruptListener()
	<-interrupt
	return nil
}

func (node *Node) HostDNS() ma.Multiaddr {
	if len(node.cfg.HostDNS) <= 0 {
		return nil
	}
	external, err := ma.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%s/p2p/%s", node.cfg.HostDNS, node.cfg.Port, node.Host().ID().String()))
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return external
}

func (node *Node) HostAddress() []string {
	hms := node.host.Addrs()
	if len(hms) <= 0 {
		return nil
	}
	result := []string{}
	for _, hm := range hms {
		result = append(result, fmt.Sprintf("%s/p2p/%s", hm.String(), node.Host().ID().String()))
	}
	return result
}

func (node *Node) startP2P() error {
	var exip string
	if len(node.cfg.ExternalIP) > 0 {
		exip = node.cfg.ExternalIP
	} else {
		eip := p2p.IpAddr()
		if eip == nil {
			return fmt.Errorf("Can't get IP")
		}
		exip = eip.String()
	}

	eMAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", exip, node.cfg.Port))
	if err != nil {
		log.Error("Unable to construct multiaddr %v", err)
		return err
	}

	srcMAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s", rconfig.DefaultIP, node.cfg.Port))
	if err != nil {
		log.Error("Unable to construct multiaddr %v", err)
		return err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrs(srcMAddr, eMAddr),
		libp2p.Identity(node.privateKey),
		libp2p.ConnectionGater(node),
	}

	if node.cfg.EnableRelay {
		opts = append(opts, libp2p.EnableRelay())
	}

	if node.cfg.HostDNS != "" {
		opts = append(opts, libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			external, err := multiaddr.NewMultiaddr(fmt.Sprintf("/dns4/%s/tcp/%s", node.cfg.HostDNS, node.cfg.Port))
			if err != nil {
				log.Error(fmt.Sprintf("Unable to create external multiaddress:%v", err))
			} else {
				addrs = append(addrs, external)
			}
			return addrs
		}))
	}

	if node.cfg.UsePeerStore {
		ps, err := node.initPeerStore()
		if err != nil {
			log.Error(err.Error())
			return err
		}
		opts = append(opts, ps)
	}

	var kademliaDHT *dht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		kademliaDHT, err = dht.New(node.Context(), h, dht.V1ProtocolOverride(p2p.ProtocolDHT()), dht.Mode(dht.ModeServer))
		return kademliaDHT, err
	}
	opts = append(opts, libp2p.Routing(newDHT))

	node.host, err = libp2p.New(opts...)
	if err != nil {
		log.Error("Failed to create host %v", err)
		return err
	}

	err = node.registerHandlers()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	err = kademliaDHT.Bootstrap(node.Context())
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("Relay Address: %s/p2p/%s\n", eMAddr.String(), node.host.ID()))
	if node.cfg.EnableRelay {
		log.Info("You can copy the relay address and configure it to the required Qitmeer-Node")
	} else {
		log.Info("The relay transport is disable.")
	}

	if len(node.cfg.HostDNS) > 0 {
		logExternalDNSAddr(node.host.ID(), node.cfg.HostDNS, node.cfg.Port)
	}

	return nil
}

func (node *Node) initPeerStore() (libp2p.Option, error) {
	dsPath := path.Join(node.cfg.DataDir, p2p.PeerStore)
	peerDS, err := ds.NewDatastore(dsPath, nil)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Start Peers from:%s", dsPath))

	ps, err := pstoreds.NewPeerstore(node.Context(), peerDS, pstoreds.DefaultOpts())
	if err != nil {
		return nil, err
	}
	return libp2p.Peerstore(ps), nil
}

func (node *Node) registerHandlers() error {

	node.host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
			remotePeer := conn.RemotePeer()
			go node.processConnected(remotePeer, conn)
		},
	})

	node.host.Network().Notify(&network.NotifyBundle{
		DisconnectedF: func(net network.Network, conn network.Conn) {
			remotePeer := conn.RemotePeer()
			go node.processDisconnected(remotePeer, conn)
		},
	})
	//

	synch.RegisterRPC(
		node,
		synch.RPCChainState,
		&pb.ChainState{},
		node.chainStateHandler,
	)

	return nil
}

func (node *Node) RegisterRpcService() error {
	if node.cfg.DisableRPC {
		return nil
	}
	cfg := node.cfg
	qcfg := &config.Config{
		DisableRPC:    cfg.DisableRPC,
		RPCListeners:  cfg.RPCListeners.Value(),
		RPCUser:       cfg.RPCUser,
		RPCPass:       cfg.RPCPass,
		RPCCert:       cfg.RPCCert,
		RPCKey:        cfg.RPCKey,
		RPCMaxClients: cfg.RPCMaxClients,
		DisableTLS:    cfg.DisableTLS,
	}

	rpcServer, err := rpc.NewRPCServer(qcfg, nil)
	if err != nil {
		return err
	}
	node.Services().RegisterService(rpcServer)
	go func() {
		<-rpcServer.RequestedProcessShutdown()
		system.ShutdownRequestChannel <- struct{}{}
	}()

	api := node.api()
	if err := rpcServer.RegisterService(api.NameSpace, api.Service); err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("RPC Service API registered. NameSpace:%s     %s", api.NameSpace, reflect.TypeOf(api.Service)))

	return nil

}

func (node *Node) RegisterBootService() error {
	if !node.cfg.Boot.Enable {
		return nil
	}
	nk, err := common.ToECDSAPrivKey(node.privateKey)
	if err != nil {
		return err
	}
	aSer, err := boot.NewBootService(node.cfg, nk)
	if err != nil {
		return err
	}
	return node.Services().RegisterService(aSer)
}

func (node *Node) Encoding() encoder.NetworkEncoding {
	return &encoder.SszNetworkEncoder{UseSnappyCompression: true}
}

func (node *Node) Host() host.Host {
	return node.host
}

func (node *Node) Disconnect(pid peer.ID) error {
	return node.host.Network().ClosePeer(pid)
}

func (node *Node) IncreaseBytesSent(pid peer.ID, size int) {
}

func (node *Node) IncreaseBytesRecv(pid peer.ID, size int) {
}

func (node *Node) chainStateHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.ChainState)
	if !ok {
		return synch.ErrMessage(fmt.Errorf("message is not type *pb.ChainState"))
	}

	pe.SetChainState(m)

	return synch.EncodeResponseMsg(node, stream, node.getChainState(), common.ErrNone)
}

func (node *Node) processConnected(pid peer.ID, conn network.Conn) {
	pe := node.peerStatus.Fetch(pid)
	node.hslock.Lock()
	defer node.hslock.Unlock()

	pe.UpdateAddrDir(nil, conn.RemoteMultiaddr(), conn.Stat().Direction)
	pe.IncreaseReConnect()
	// Handle the various pre-existing conditions that will result in us not handshaking.
	if pe.IsConnected() {
		log.Trace(fmt.Sprintf("%s currentState:%s reason:already connected, Ignoring connection request", pe.IDWithAddress(), pe.ConnectionState().String()))
		return
	}

	pe.SetConnectionState(peers.PeerConnected)
	// Go through the handshake process.
	multiAddr := fmt.Sprintf("%s/p2p/%s", pe.Address().String(), pe.GetID().String())

	log.Info(fmt.Sprintf("%s direction:%s multiAddr:%s",
		pe.GetID(), pe.Direction(), multiAddr))
}

func (node *Node) processDisconnected(pid peer.ID, conn network.Conn) {
	node.hslock.Lock()
	defer node.hslock.Unlock()

	pe := node.peerStatus.Get(pid)
	if pe == nil {
		return
	}
	if pe.ConnectionState().IsDisconnected() {
		return
	}
	pe.SetConnectionState(peers.PeerDisconnected)
	log.Info(fmt.Sprintf("%s Peer Disconnected", pe.IDWithAddress()))
}

func (node *Node) getChainState() *pb.ChainState {
	genesisHash := params.ActiveNetParams.GenesisHash

	gs := &pb.GraphState{
		Total:      1,
		Layer:      0,
		MainHeight: 0,
		MainOrder:  0,
		Tips:       []*pb.Hash{},
	}
	gs.Tips = append(gs.Tips, &pb.Hash{Hash: genesisHash.Bytes()})

	return &pb.ChainState{
		GenesisHash:     &pb.Hash{Hash: genesisHash.Bytes()},
		ProtocolVersion: pv.ProtocolVersion,
		Timestamp:       uint64(roughtime.Now().Unix()),
		Services:        uint64(pv.Relay),
		GraphState:      gs,
		UserAgent:       []byte(p2p.BuildUserAgent("Qitmeer-relay")),
		DisableRelayTx:  true,
	}
}

func (node *Node) isPeerAtLimit() bool {
	numOfConns := len(node.host.Network().Peers())
	maxPeers := int(node.cfg.MaxPeers)
	activePeers := len(node.peerStatus.Active())

	return activePeers >= maxPeers || numOfConns >= maxPeers
}

// InterceptPeerDial tests whether we're permitted to Dial the specified peer.
func (node *Node) InterceptPeerDial(p peer.ID) (allow bool) {
	if node.isPeerAtLimit() {
		log.Trace(fmt.Sprintf("peer:%s reason:at peer max limit", p.String()))
		return false
	}
	return true
}

// InterceptAddrDial tests whether we're permitted to dial the specified
// multiaddr for the given peer.
func (node *Node) InterceptAddrDial(_ peer.ID, m multiaddr.Multiaddr) (allow bool) {
	if node.isPeerAtLimit() {
		log.Trace(fmt.Sprintf("peer:%s reason:at peer max limit", m.String()))
		return false
	}
	return true
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
func (node *Node) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	if node.isPeerAtLimit() {
		log.Trace(fmt.Sprintf("peer:%s reason:at peer max limit", n.RemoteMultiaddr().String()))
		return false
	}
	return true
}

// InterceptSecured tests whether a given connection, now authenticated,
// is allowed.
func (node *Node) InterceptSecured(_ network.Direction, _ peer.ID, n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
func (node *Node) InterceptUpgraded(n network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

func (node *Node) GetRpcServer() *rpc.RpcServer {
	var service *rpc.RpcServer
	if err := node.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (node *Node) GetBootService() *boot.BootService {
	var service *boot.BootService
	if err := node.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (node *Node) Peers() *peers.Status {
	return node.peerStatus
}

func (node *Node) IsRunning() bool {
	return !node.IsShutdown() && node.IsStarted()
}
func (node *Node) GetGenesisHash() *hash.Hash {
	return params.ActiveNetParams.GenesisHash
}

func logExternalDNSAddr(id peer.ID, addr string, port string) {
	if addr != "" {
		log.Info(fmt.Sprintf("Relay node started external p2p server:multiAddr=%s", "/dns4/"+addr+"/tcp/"+port+"/p2p/"+id.String()))
	}
}
