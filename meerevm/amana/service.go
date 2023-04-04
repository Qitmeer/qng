package amana

import (
	"fmt"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	mconsensus "github.com/Qitmeer/qng/meerevm/amana/consensus"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethclient"
	"os"
	"path"
)

type AmanaService struct {
	service.Service
	cfg   *config.Config
	cons  model.Consensus
	chain *eth.ETHChain
}

func (q *AmanaService) Start() error {
	if err := q.Service.Start(); err != nil {
		return err
	}
	log.Info("Start AmanaService")

	ecfg, args, err := MakeParams(q.cfg)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	chain, err := eth.NewETHChain(ecfg, args)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	q.chain = chain
	//

	if chain.Context().Bool(utils.MiningEnabledFlag.Name) || chain.Context().Bool(utils.DeveloperFlag.Name) {
		eb, err := chain.Ether().Etherbase()
		if err != nil {
			return fmt.Errorf("etherbase missing: %v", err)
		}
		wallet, err := chain.Ether().AccountManager().Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		chain.Ether().Engine().(*mconsensus.Amana).Authorize(eb, wallet.SignData)
		log.Info(fmt.Sprintf("Amana Authorize:%s", eb))
	}
	//
	err = q.chain.Start()
	if err != nil {
		return err
	}
	//
	rpcClient, err := q.chain.Node().Attach()
	if err != nil {
		log.Error(fmt.Sprintf("Failed to attach to self: %v", err))
	}
	client := ethclient.NewClient(rpcClient)

	blockNum, err := client.BlockNumber(q.Context())
	if err != nil {
		log.Error(err.Error())
	} else {
		log.Debug(fmt.Sprintf("Amana block chain current block number:%d", blockNum))
	}

	cbh := q.chain.Ether().BlockChain().CurrentBlock()
	if cbh != nil {
		log.Debug(fmt.Sprintf("Amana block chain current block:number=%d hash=%s", cbh.Number.Uint64(), cbh.Hash().String()))
	}

	//
	state, err := q.chain.Ether().BlockChain().State()
	if err != nil {
		return nil
	}
	//
	for addr := range q.chain.Config().Eth.Genesis.Alloc {
		log.Debug(fmt.Sprintf("Amana Alloc address:%v balance:%v", addr.String(), state.GetBalance(addr)))
	}
	return nil
}

func (q *AmanaService) Stop() error {
	if err := q.Service.Stop(); err != nil {
		return err
	}
	return q.chain.Stop()
}

func (q *AmanaService) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicAmanaServiceAPI(q),
			Public:    true,
		},
	}
}

// TODO: Just compatible with old data, will be deleted in the future
func (q *AmanaService) Upgrade() error {
	oldDBPath := path.Join(q.cfg.DataDir, mconsensus.IdentifierDeprecated)
	if !util.FileExists(oldDBPath) {
		return nil
	}
	log.Info("Discover old data and prepare for compatible operations:", mconsensus.IdentifierDeprecated, ClientIdentifier)
	return os.Rename(oldDBPath, path.Join(q.cfg.DataDir, ClientIdentifier))
}

func New(cfg *config.Config, cons model.Consensus) (*AmanaService, error) {
	a := AmanaService{
		cfg:  cfg,
		cons: cons,
	}
	return &a, nil
}
