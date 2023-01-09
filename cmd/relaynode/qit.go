package main

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"net"
)

type QitConfig struct {
	Enable      bool
	listenAddr  string
	natdesc     string
	netrestrict string
	runv5       bool
}

type QitService struct {
	service.Service
	cfg       *Config
	nodeKey   *ecdsa.PrivateKey
	localNode *enode.LocalNode
}

func (s *QitService) Start() error {
	if err := s.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Start Qit Service ..."))

	eth.InitLog(s.cfg.DebugLevel, s.cfg.DebugPrintOrigins)

	var err error
	var natm nat.Interface
	if len(s.cfg.Qit.natdesc) > 0 {
		natm, err = nat.Parse(s.cfg.Qit.natdesc)
		if err != nil {
			return fmt.Errorf("--nat: %v", err)
		}
	}

	var restrictList *netutil.Netlist
	if len(s.cfg.Qit.netrestrict) > 0 {
		restrictList, err = netutil.ParseNetlist(s.cfg.Qit.netrestrict)
		if err != nil {
			return fmt.Errorf("--netrestrict: %v", err)
		}
	}

	addr, err := net.ResolveUDPAddr("udp", s.cfg.Qit.listenAddr)
	if err != nil {
		return fmt.Errorf("ResolveUDPAddr: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("ListenUDP: %v", err)
	}
	realaddr := conn.LocalAddr().(*net.UDPAddr)
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "QitSubnet discovery")
		}
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	db, _ := enode.OpenDB("")
	s.localNode = enode.NewLocalNode(db, s.nodeKey)

	log.Info(fmt.Sprintf("QitSubnet bootnode:%s", s.localNode.Node().URLv4()))
	log.Info(fmt.Sprintf("QitSubnet bootnode:%s", s.localNode.Node().String()))

	cfg := discover.Config{
		PrivateKey:  s.nodeKey,
		NetRestrict: restrictList,
	}
	if s.cfg.Qit.runv5 {
		if _, err := discover.ListenV5(conn, s.localNode, cfg); err != nil {
			return err
		}
	} else {
		if _, err := discover.ListenUDP(conn, s.localNode, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *QitService) Stop() error {
	if err := s.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop Qit Service"))
	return nil
}

func (s *QitService) Node() *enode.LocalNode {
	return s.localNode
}

func NewQitService(cfg *Config, nodeKey *ecdsa.PrivateKey) (*QitService, error) {
	return &QitService{
		cfg:     cfg,
		nodeKey: nodeKey,
	}, nil
}
