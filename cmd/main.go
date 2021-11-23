/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package main

import (
	"github.com/Qitmeer/meerevm/evm"
	"github.com/Qitmeer/meerevm/evm/util"
	"github.com/ethereum/go-ethereum/log"
)

func main() {
	util.InitLog("trace",true)
	log.Info(evm.New().Version())
}
