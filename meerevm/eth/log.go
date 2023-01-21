/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package eth

import (
	qlog "github.com/Qitmeer/qng/log"
	"github.com/ethereum/go-ethereum/log"
)

var glogger *log.GlogHandler

func InitLog(DebugLevel string, DebugPrintOrigins bool) {
	if glogger == nil {
		glogger = log.NewGlogHandler(log.StreamHandler(qlog.LogWrite(), TerminalFormat(qlog.LogWrite().IsUseColor())))
		glogger.Verbosity(log.LvlTrace)
		log.Root().SetHandler(glogger)
	}

	lvl, err := log.LvlFromString(DebugLevel)
	if err == nil {
		glogger.Verbosity(lvl)
	}
	log.PrintOrigins(DebugPrintOrigins)

	qlog.LocationTrims = append(qlog.LocationTrims, "github.com/ethereum/go-ethereum/")
}

func TerminalFormat(usecolor bool) log.Format {
	logTF := qlog.TerminalFormat(usecolor)
	return log.FormatFunc(func(r *log.Record) []byte {
		qr := &qlog.Record{
			Time:     r.Time,
			Lvl:      qlog.Lvl(r.Lvl),
			Msg:      r.Msg,
			Ctx:      r.Ctx,
			Call:     r.Call,
			KeyNames: qlog.RecordKeyNames{Time: r.KeyNames.Time, Msg: r.KeyNames.Msg, Lvl: r.KeyNames.Lvl},
		}
		return logTF.Format(qr)
	})
}
