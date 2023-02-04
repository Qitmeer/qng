package main

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/Qitmeer/qng/cmd/relaynode/config"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

type QitDNSService struct {
	service.Service
	cfg       *config.Config
	nodeKey   *ecdsa.PrivateKey
	localNode *enode.Node
}

func (s *QitDNSService) Start() error {
	if err := s.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Start Qit DNS Service ..."))

	eth.InitLog(s.cfg.DebugLevel, s.cfg.DebugPrintOrigins)

	return nil
}

func (s *QitDNSService) Stop() error {
	if err := s.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop Qit DNS Service"))
	return nil
}

func (s *QitDNSService) Node() *enode.Node {
	return s.localNode
}

func NewQitDNSService(cfg *config.Config, nodeKey *ecdsa.PrivateKey) (*QitDNSService, error) {
	return &QitDNSService{
		cfg:     cfg,
		nodeKey: nodeKey,
	}, nil
}
