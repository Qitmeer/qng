// Copyright (c) 2017-2018 The qitmeer developers
package node

import (
	"fmt"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/common/system/disk"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/coinbase"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/meerevm/amana"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/p2p"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/services/acct"
	"github.com/Qitmeer/qng/services/address"
	"github.com/Qitmeer/qng/services/mempool"
	"github.com/Qitmeer/qng/services/miner"
	"github.com/Qitmeer/qng/services/mining"
	"github.com/Qitmeer/qng/services/notifymgr"
	"github.com/Qitmeer/qng/services/tx"
	ecommon "github.com/ethereum/go-ethereum/common"
	"path/filepath"
	"reflect"
	"time"
)

// QitmeerFull implements the qitmeer full node service.
type QitmeerFull struct {
	service.Service
	// under node
	node *Node
	// msg notifier
	nfManager model.Notify
	// database
	db model.DataBase
	// address service
	addressApi *address.AddressApi
}

func (qm *QitmeerFull) APIs() []api.API {
	apis := qm.Service.APIs()
	apis = append(apis, qm.addressApi.APIs()...)
	apis = append(apis, qm.apis()...)
	return apis
}

func (qm *QitmeerFull) RegisterP2PService() error {
	peerServer, err := p2p.NewService(qm.node.Config, qm.node.consensus, qm.node.Params)
	if err != nil {
		return err
	}
	return qm.Services().RegisterService(peerServer)
}

func (qm *QitmeerFull) RegisterRpcService() ([]api.API, error) {
	if qm.node.Config.DisableRPC {
		return nil, nil
	}
	rpcServer, err := rpc.NewRPCServer(qm.node.Config, qm.node.consensus)
	if err != nil {
		return nil, err
	}
	qm.Services().RegisterService(rpcServer)

	go func() {
		<-rpcServer.RequestedProcessShutdown()
		system.ShutdownRequestChannel <- struct{}{}
	}()
	// Gather all the possible APIs to surface
	apis := qm.APIs()

	// Generate the whitelist based on the allowed modules
	whitelist := make(map[string]bool)
	for _, module := range qm.node.Config.Modules {
		whitelist[module] = true
	}

	retApis := []api.API{}
	// Register all the APIs exposed by the services
	for _, api := range apis {
		if whitelist[api.NameSpace] || (len(whitelist) == 0 && api.Public) {
			if err := rpcServer.RegisterService(api.NameSpace, api.Service); err != nil {
				return nil, err
			}
			log.Debug(fmt.Sprintf("RPC Service API registered. NameSpace:%s     %s", api.NameSpace, reflect.TypeOf(api.Service)))
			retApis = append(retApis, api)
		}
	}
	return retApis, nil
}

func (qm *QitmeerFull) RegisterTxManagerService() error {
	// txmanager
	tm, err := tx.NewTxManager(qm.node.consensus, qm.nfManager)
	if err != nil {
		return err
	}
	qm.Services().RegisterService(tm)
	return nil
}

func (qm *QitmeerFull) RegisterMinerService() error {
	cfg := qm.node.Config
	txManager := qm.GetTxManager()
	// Cpu Miner
	// Create the mining policy based on the configuration options.
	// NOTE: The CPU miner relies on the mempool, so the mempool has to be
	// created before calling the function to create the CPU miner.
	policy := mining.Policy{
		BlockMinSize:      cfg.BlockMinSize,
		BlockMaxSize:      cfg.BlockMaxSize,
		BlockPrioritySize: cfg.BlockPrioritySize,
		TxMinFreeFee:      cfg.MinTxFee, //TODO, duplicated config item with mem-pool
		TxTimeScope:       cfg.TxTimeScope,
		StandardVerifyFlags: func() (txscript.ScriptFlags, error) {
			return mempool.StandardScriptVerifyFlags()
		}, //TODO, duplicated config item with mem-pool
		CoinbaseGenerator: coinbase.NewCoinbaseGenerator(qm.node.Params, qm.GetPeerServer().PeerID().String()),
	}
	miner := miner.NewMiner(qm.node.consensus, &policy, txManager.MemPool().(*mempool.TxPool), qm.GetPeerServer())
	qm.Services().RegisterService(miner)
	return nil
}

func (qm *QitmeerFull) RegisterNotifyMgr() error {
	nfManager := notifymgr.New(qm.GetPeerServer(), qm.node.consensus)
	qm.Services().RegisterService(nfManager)
	qm.nfManager = nfManager
	return nil
}

func (qm *QitmeerFull) RegisterAccountService(cfg *config.Config) error {
	// account manager
	acctmgr, err := acct.New(qm.GetBlockChain(), cfg)
	if err != nil {
		return err
	}
	qm.GetBlockChain().Acct = acctmgr
	qm.Services().RegisterService(acctmgr)
	return nil
}

func (qm *QitmeerFull) RegisterAmana() error {
	if !qm.node.Config.Amana ||
		params.ActiveNetParams.Net == protocol.MainNet {
		return nil
	}
	ser, err := amana.New(qm.node.Config, qm.node.consensus)
	if err != nil {
		return err
	}
	return qm.Services().RegisterService(ser)
}

// return address api
func (qm *QitmeerFull) GetAddressApi() *address.AddressApi {
	return qm.addressApi
}

// return peer server
func (qm *QitmeerFull) GetPeerServer() *p2p.Service {
	var service *p2p.Service
	if err := qm.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (qm *QitmeerFull) GetRpcServer() *rpc.RpcServer {
	var service *rpc.RpcServer
	if err := qm.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (qm *QitmeerFull) GetTxManager() *tx.TxManager {
	var service *tx.TxManager
	if err := qm.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (qm *QitmeerFull) GetMiner() *miner.Miner {
	var service *miner.Miner
	if err := qm.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (qm *QitmeerFull) GetBlockChain() *blockchain.BlockChain {
	var service *blockchain.BlockChain
	if err := qm.Services().FetchService(&service); err != nil {
		log.Error(err.Error())
		return nil
	}
	return service
}

func (qm *QitmeerFull) monitorFreeDiskSpace() error {
	freeDiskSpaceCritical := qm.node.Config.Minfreedisk * 1024 * 1024
	if freeDiskSpaceCritical == 0 {
		return nil
	}
	path, err := filepath.Abs(qm.node.Config.DataDir)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("monitor disk path is empty")
	}
	go func() {
		log.Info("Start monitor free disk space", "path", path, "critical", ecommon.StorageSize(freeDiskSpaceCritical))
		for {
			freeSpace, err := disk.GetFreeDiskSpace(path)
			if err != nil {
				log.Warn("Failed to get free disk space", "path", path, "err", err)
				break
			}
			if freeSpace < freeDiskSpaceCritical {
				log.Error("Low disk space. Gracefully shutting down QNG to prevent database corruption.", "available", ecommon.StorageSize(freeSpace), "path", path)
				qm.node.consensus.Shutdown()
				break
			} else if freeSpace < 2*freeDiskSpaceCritical {
				log.Warn("Disk space is running low. QNG will shutdown if disk space runs below critical level.", "available", ecommon.StorageSize(freeSpace), "critical_level", ecommon.StorageSize(freeDiskSpaceCritical), "path", path)
			}
			time.Sleep(30 * time.Second)
		}
	}()

	return nil
}

func newQitmeerFullNode(node *Node) (*QitmeerFull, error) {
	qm := QitmeerFull{
		node: node,
		db:   node.DB,
	}
	qm.Service.InitServices()

	cfg := node.Config

	if err := node.consensus.Init(); err != nil {
		return nil, err
	}
	bc := node.consensus.BlockChain().(*blockchain.BlockChain)
	if err := qm.Services().RegisterService(bc); err != nil {
		return nil, err
	}
	if err := qm.RegisterP2PService(); err != nil {
		return nil, err
	}
	if err := qm.RegisterNotifyMgr(); err != nil {
		return nil, err
	}
	if err := qm.RegisterTxManagerService(); err != nil {
		return nil, err
	}

	txManager := qm.GetTxManager()
	// prepare peerServer
	qm.GetPeerServer().SetBlockChain(qm.GetBlockChain())
	qm.GetPeerServer().SetTimeSource(qm.node.consensus.MedianTimeSource())
	qm.GetPeerServer().SetTxMemPool(txManager.MemPool().(*mempool.TxPool))
	qm.GetPeerServer().SetNotify(qm.nfManager)

	//
	bc.MeerChain().(*meer.MeerChain).MeerPool().SetTxPool(txManager.MemPool())
	bc.MeerChain().(*meer.MeerChain).MeerPool().SetNotify(qm.nfManager)
	//
	if err := qm.RegisterMinerService(); err != nil {
		return nil, err
	}
	// init address api
	qm.addressApi = address.NewAddressApi(cfg, node.Params, qm.GetBlockChain())

	if err := qm.RegisterAccountService(cfg); err != nil {
		return nil, err
	}

	if qm.node.consensus.AmanaService() != nil {
		if err := qm.Services().RegisterService(qm.node.consensus.AmanaService()); err != nil {
			return nil, err
		}
	}

	apis, err := qm.RegisterRpcService()
	if err != nil {
		return nil, err
	}
	bc.MeerChain().RegisterAPIs(apis)

	if qm.GetRpcServer() != nil {
		qm.GetRpcServer().BC = qm.GetBlockChain()
		qm.GetRpcServer().ChainParams = qm.node.Params

		qm.nfManager.(*notifymgr.NotifyMgr).RpcServer = qm.GetRpcServer()
		qm.GetMiner().RpcSer = qm.GetRpcServer()
	}

	qm.Services().LowestPriority(qm.GetBlockChain())
	qm.Services().LowestPriority(qm.GetTxManager())
	qm.Services().LowestPriority(qm.GetPeerServer())
	return &qm, qm.monitorFreeDiskSpace()
}
