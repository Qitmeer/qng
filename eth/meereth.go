// (c) 2021, the Qitmeer developers. All rights reserved.
// license that can be found in the LICENSE file.

// Package meereth encapsulated the Ethereum protocol.
package meereth

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"io"
	"path/filepath"
)

type Ether struct {
	Backend *eth.Ethereum
}

type Key struct {
	Address    common.Address
	PrivateKey *ecdsa.PrivateKey
}

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *Key {
	key := &Key{
		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

func NewKey(rand io.Reader) (*Key, error) {
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
	if err != nil {
		return nil, err
	}
	return NewKeyFromECDSA(privateKeyECDSA), nil
}

type Config struct {
	EthConfig *ethconfig.Config
}

// clientIdentifier is a hard coded identifier to report into the network.
var clientIdentifier = "MeerEth"

func New(config *Config) (*node.Node, *Ether) {

	datadir := "./data"
	fpDataDir, err := filepath.Abs(datadir)
	if err != nil {
		return nil, nil
	}

	ecethash := ethconfig.Defaults.Ethash
	ecethash.DatasetDir = filepath.Join(fpDataDir, "dataset")
	ecethash.PowMode = ethash.ModeFullFake
	config.EthConfig.Ethash = ecethash

	// Create the empty networking stack
	nodeConf := &node.Config{
		Name:                clientIdentifier,
		Version:             params.VersionWithMeta,
		DataDir:             datadir,
		KeyStoreDir:         filepath.Join(datadir, "keystore"),
		HTTPPort:            node.DefaultHTTPPort,
		HTTPModules:         []string{"net", "web3"},
		HTTPVirtualHosts:    []string{"localhost"},
		HTTPTimeouts:        rpc.DefaultHTTPTimeouts,
		WSPort:              node.DefaultWSPort,
		WSModules:           []string{"net", "web3"},
		GraphQLVirtualHosts: []string{"localhost"},
		P2P: p2p.Config{
			MaxPeers:    0,
			DiscoveryV5: false,
			NoDiscovery: true,
			NoDial:      true,
		},
	}

	stack, err := node.New(nodeConf)
	if err != nil {
		utils.Fatalf("Failed to create the node: %v", err)
	}

	backend, _ := eth.New(stack, config.EthConfig)
	if err != nil {
		utils.Fatalf("Failed to create the eth backend: %v", err)
	}
	ether := &Ether{Backend: backend}

	if err != nil {
		utils.Fatalf("failed to start node: %v", err)
	}
	return stack, ether
}
