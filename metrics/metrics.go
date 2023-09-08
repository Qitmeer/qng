// Copyright (c) 2017-2019 The Qitmeer developers
//
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// The parts code inspired & originated from
// https://github.com/ethereum/go-ethereum/metrics

// Package metrics provides general system and process level metrics collection.
package metrics

import (
	"github.com/Qitmeer/qng/log"
	emetrics "github.com/ethereum/go-ethereum/metrics"
	"os"
	"strings"
)

// Init enables or disables the metrics system. Since we need this to run before
// any other code gets to create meters and timers, we'll actually do an ugly hack
// and peek into the command line args for the metrics flag.

func init() {
	// enablerFlags is the CLI flag names to use to enable metrics collections.
	var enablerFlags = []string{"metrics"}
	// expensiveEnablerFlags is the CLI flag names to use to enable metrics collections.
	var expensiveEnablerFlags = []string{"metrics.expensive"}
	for _, arg := range os.Args {
		flag := strings.TrimLeft(arg, "-")
		for _, enabler := range enablerFlags {
			if !emetrics.Enabled && flag == enabler {
				log.Info("Enabling metrics collection.")
				emetrics.Enabled = true
			}
		}
		for _, enabler := range expensiveEnablerFlags {
			if !emetrics.EnabledExpensive && flag == enabler {
				log.Info("Enabling expensive metrics collection.")
				emetrics.EnabledExpensive = true
			}
		}
	}
}
