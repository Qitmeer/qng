/*
 * Copyright (c) 2020.
 * Project:qitmeer
 * File:binode.go
 * Date:6/6/20 9:28 AM
 * Author:Jin
 * Email:lochjin@gmail.com
 */

package main

import (
	"fmt"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/services/common"
	"path"
)

type BINode struct {
	name string
	bc   *blockchain.BlockChain
	db   database.DB
	cfg  *Config
}

func (node *BINode) init(cfg *Config) error {
	node.cfg = cfg

	// Load the block database.
	db, err := LoadBlockDB(cfg.DbType, cfg.DataDir, true)
	if err != nil {
		log.Error("load block database", "error", err)
		return err
	}

	node.db = db
	//
	ccfg := common.DefaultConfig(node.cfg.HomeDir)
	ccfg.DataDir = cfg.DataDir
	ccfg.DbType = cfg.DbType
	ccfg.DAGType = cfg.DAGType
	cons := consensus.NewPure(ccfg, db)
	err = cons.Init()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	node.bc = cons.BlockChain().(*blockchain.BlockChain)
	node.name = path.Base(cfg.DataDir)

	log.Info(fmt.Sprintf("Load Data:%s", cfg.DataDir))

	return node.statistics()
}

func (node *BINode) exit() {
	if node.db != nil {
		log.Info(fmt.Sprintf("Gracefully shutting down the database:%s", node.name))
		node.db.Close()
	}
}

func (node *BINode) BlockChain() *blockchain.BlockChain {
	return node.bc
}

func (node *BINode) DB() database.DB {
	return node.db
}

func (node *BINode) statistics() error {
	total := node.bc.BlockDAG().GetBlockTotal()
	validCount := 1
	subsidyCount := 0
	subsidy := uint64(0)
	fmt.Printf("Process...   ")
	for i := uint(1); i < total; i++ {
		ib := node.bc.BlockDAG().GetBlockById(i)
		if ib == nil {
			return fmt.Errorf("No block:%d", i)
		}
		if !knownInvalid(byte(ib.GetState().GetStatus())) {
			validCount++

			block, err := node.bc.FetchBlockByHash(ib.GetHash())
			if err != nil {
				return err
			}

			txfullHash := block.Transactions()[0].Tx.TxHashFull()

			if isTxValid(node.db, block.Transactions()[0].Hash(), &txfullHash, ib.GetHash()) {
				if node.bc.BlockDAG().IsBlue(i) {
					subsidyCount++
					subsidy += uint64(block.Transactions()[0].Tx.TxOut[0].Amount.Value)
				}
			}
		}

	}
	mainTip := node.bc.BlockDAG().GetMainChainTip().(*meerdag.PhantomBlock)
	blues := mainTip.GetBlueNum() + 1
	reds := mainTip.GetOrder() + 1 - blues
	unconfirmed := total - (mainTip.GetOrder() + 1)

	fmt.Println()
	fmt.Printf("Total:%d   Valid:%d   BlueNum:%d   RedNum:%d   SubsidyNum:%d Subsidy:%d", total, validCount, blues, reds, subsidyCount, subsidy)
	if unconfirmed > 0 {
		fmt.Printf(" Unconfirmed:%d", unconfirmed)
	}
	fmt.Println()
	fmt.Println("(Note:SubsidyNum does not include genesis.)")

	return nil
}
