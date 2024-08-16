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
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"path"
)

type Node struct {
	name     string
	bc       *blockchain.BlockChain
	db       legacydb.DB
	cfg      *Config
	endPoint meerdag.IBlock
}

func (node *Node) init(cfg *Config, srcnode *SrcNode, endPoint meerdag.IBlock) error {
	node.cfg = cfg
	node.endPoint = endPoint
	//
	qcfg := cfg.ToQNGConfig()
	database.Cleanup(qcfg)
	// Load the block database.
	db, err := database.New(qcfg, system.InterruptListener())
	if err != nil {
		log.Error("load block database", "error", err)
		return err
	}

	node.db = db.(*legacychaindb.LegacyChainDB).DB()
	cons := consensus.NewPure(qcfg, db)
	err = cons.Init()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	node.bc = cons.BlockChain().(*blockchain.BlockChain)
	node.name = path.Base(cfg.DataDir)

	log.Info(fmt.Sprintf("Load Data:%s", cfg.DataDir))

	return node.processBlockDAG(srcnode)
}

func (node *Node) exit() {
	if node.db != nil {
		log.Info(fmt.Sprintf("Gracefully shutting down the database:%s", node.name))
		node.db.Close()
	}
}

func (node *Node) BlockChain() *blockchain.BlockChain {
	return node.bc
}

func (node *Node) DB() legacydb.DB {
	return node.db
}

func (node *Node) processBlockDAG(srcnode *SrcNode) error {
	genesisHash := node.bc.BlockDAG().GetGenesisHash()
	srcgenesisHash := srcnode.BlockChain().BlockDAG().GetGenesisHash()
	if !genesisHash.IsEqual(srcgenesisHash) {
		return fmt.Errorf("Different genesis!")
	}
	srcTotal := srcnode.bc.BlockDAG().GetBlockTotal()
	if node.endPoint.GetHash().IsEqual(genesisHash) {
		return nil
	}

	log.Glogger().Verbosity(log.LvlCrit)
	var bar *ProgressBar
	i := uint(1)
	if !node.cfg.DisableBar {

		bar = &ProgressBar{}
		bar.init("Process:")
		bar.reset(int(node.endPoint.GetID() + 1))
		bar.add()
	} else {
		log.Info("Process...")
	}

	defer func() {
		log.Glogger().Verbosity(log.LvlInfo)
		if bar != nil {
			fmt.Println()
		}
		log.Info(fmt.Sprintf("End process block DAG:(%d/%d)", i-1, srcTotal))
	}()
	mainTip := srcnode.bc.BlockDAG().GetMainChainTip()
	for ; i < mainTip.GetID(); i++ {
		ib := srcnode.bc.BlockDAG().GetBlockById(i)
		if ib == nil {
			return fmt.Errorf("Can't find block id (%d)!", i)
		}

		block, err := srcnode.bc.FetchBlockByHash(ib.GetHash())
		if err != nil {
			return err
		}
		err = node.bc.CheckBlockSanity(block, node.bc.TimeSource(), blockchain.BFFastAdd, params.ActiveNetParams.Params)
		if err != nil {
			return err
		}
		_, err = node.bc.FastAcceptBlock(block, blockchain.BFFastAdd)
		if err != nil {
			return err
		}
		if bar != nil {
			bar.add()
		}
		if ib.GetHash().IsEqual(node.endPoint.GetHash()) {
			break
		}
	}
	if bar != nil {
		bar.setMax()
	}
	return nil
}
