/*
 * Copyright (c) 2020.
 * Project:qitmeer
 * File:srcnode.go
 * Date:5/13/20 6:45 AM
 * Author:Jin
 * Email:lochjin@gmail.com
 */

package main

import (
	"fmt"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/database/legacychaindb"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/log"
	"path"
)

type SrcNode struct {
	name string
	bc   *blockchain.BlockChain
	db   legacydb.DB
	cfg  *Config
}

func (node *SrcNode) init(cfg *Config) error {
	node.cfg = cfg
	tempCfg := *cfg
	// Load the block database.
	tempCfg.DataDir = cfg.SrcDataDir
	if cfg.Last {
		tempCfg.DataDir = cfg.DataDir
	}
	tempQCfg := tempCfg.ToQNGConfig()
	legacychaindb.CreateIfNoExist = false
	db, err := database.New(tempQCfg, system.InterruptListener())
	if err != nil {
		log.Error("load block database", "error", err)
		return err
	}
	node.db = db.(*legacychaindb.LegacyChainDB).DB()
	//
	cons := consensus.NewPure(cfg.ToQNGConfig(), db)
	err = cons.Init()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	node.bc = cons.BlockChain().(*blockchain.BlockChain)
	node.name = path.Base(tempCfg.DataDir)

	log.Info(fmt.Sprintf("Load Src Data:%s", tempCfg.DataDir))
	return nil
}

func (node *SrcNode) exit() {
	if node.db != nil {
		log.Info(fmt.Sprintf("Gracefully shutting down the database:%s", node.name))
		node.db.Close()
	}
}

func (node *SrcNode) BlockChain() *blockchain.BlockChain {
	return node.bc
}

func (node *SrcNode) DB() legacydb.DB {
	return node.db
}
