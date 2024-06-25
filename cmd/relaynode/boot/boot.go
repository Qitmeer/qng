package boot

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"net"
)

type BootService struct {
	service.Service
	cfg       *config.Config
	nodeKey   *ecdsa.PrivateKey
	localNode *enode.Node
}

func (s *BootService) Start() error {
	if err := s.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Start Boot Service ..."))

	eth.InitLog(s.cfg.DebugLevel, s.cfg.DebugPrintOrigins)

	var err error
	var natm nat.Interface
	if len(s.cfg.Boot.Natdesc) > 0 {
		natm, err = nat.Parse(s.cfg.Boot.Natdesc)
		if err != nil {
			return fmt.Errorf("--nat: %v", err)
		}
	}

	var restrictList *netutil.Netlist
	if len(s.cfg.Boot.Netrestrict) > 0 {
		restrictList, err = netutil.ParseNetlist(s.cfg.Boot.Netrestrict)
		if err != nil {
			return fmt.Errorf("--netrestrict: %v", err)
		}
	}

	addr, err := net.ResolveUDPAddr("udp", s.cfg.Boot.ListenAddr)
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
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "Boot discovery")
		}
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	s.setLocalNode(&s.nodeKey.PublicKey, *realaddr)

	db, _ := enode.OpenDB("")
	ln := enode.NewLocalNode(db, s.nodeKey)
	cfg := discover.Config{
		PrivateKey:  s.nodeKey,
		NetRestrict: restrictList,
	}
	if s.cfg.Boot.Runv5 {
		if _, err := discover.ListenV5(conn, ln, cfg); err != nil {
			return err
		}
	} else {
		if _, err := discover.ListenUDP(conn, ln, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *BootService) Stop() error {
	if err := s.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop Boot Service"))
	return nil
}

func (s *BootService) Node() *enode.Node {
	return s.localNode
}

func (s *BootService) setLocalNode(nodeKey *ecdsa.PublicKey, addr net.UDPAddr) {
	if addr.IP.IsUnspecified() {
		addr.IP = net.IP{127, 0, 0, 1}
	}
	s.localNode = enode.NewV4(nodeKey, addr.IP, 0, addr.Port)
	log.Info(fmt.Sprintf("Boot:%s", s.localNode.URLv4()))
}

func NewBootService(cfg *config.Config, nodeKey *ecdsa.PrivateKey) (*BootService, error) {
	return &BootService{
		cfg:     cfg,
		nodeKey: nodeKey,
	}, nil
}
