/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chain

import (
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/node"
)

type MeerethstatsConfig struct {
	URL string `toml:",omitempty"`
}

type MeerethConfig struct {
	Eth      ethconfig.Config
	Node     node.Config
	Ethstats MeerethstatsConfig
	Metrics  metrics.Config
}