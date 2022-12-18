/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package main

import (
	"context"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/meerevm/evm"
	"github.com/Qitmeer/qng/meerevm/evm/util"
	"github.com/Qitmeer/qng/vm/consensus"
	"github.com/ethereum/go-ethereum/log"
)

type MContext struct {
	context.Context
	Cfg *config.Config
}

func (ctx *MContext) GetConfig() *config.Config {
	return ctx.Cfg
}

func (ctx *MContext) GetTxPool() model.TxPool {
	return nil
}

func (ctx *MContext) GetNotify() consensus.Notify {
	return nil
}

func main() {
	debugLevel := "trace"
	debugPrintOrigins := true
	util.InitLog(debugLevel, debugPrintOrigins)

	ctx := &MContext{
		Context: context.Background(),
		Cfg: &config.Config{
			DataDir:           "./data",
			DebugLevel:        debugLevel,
			DebugPrintOrigins: debugPrintOrigins,
		},
	}
	vm := evm.New()
	vm.Initialize(ctx)
	log.Info(vm.Version())
}
