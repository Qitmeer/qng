/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package util

import (
	"github.com/ethereum/go-ethereum/log"
	"os"
	qlog "github.com/Qitmeer/qng-core/log"
)

func InitLog(DebugLevel string,DebugPrintOrigins bool)  {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlTrace)
	log.Root().SetHandler(glogger)

	lvl, err := log.LvlFromString(DebugLevel)
	if err == nil {
		glogger.Verbosity(lvl)
	}
	log.PrintOrigins(DebugPrintOrigins)

	qlog.LocationTrims=append(qlog.LocationTrims,"github.com/Qitmeer/meerevm")
}
