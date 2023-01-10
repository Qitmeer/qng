/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	"github.com/ethereum/go-ethereum/log"
	"os"
)

var glogger *log.GlogHandler

func InitLog(DebugLevel string, DebugPrintOrigins bool) {
	if glogger == nil {
		glogger = log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
		glogger.Verbosity(log.LvlTrace)
		log.Root().SetHandler(glogger)
	}

	lvl, err := log.LvlFromString(DebugLevel)
	if err == nil {
		glogger.Verbosity(lvl)
	}
	log.PrintOrigins(DebugPrintOrigins)
}
