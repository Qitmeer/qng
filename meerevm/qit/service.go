package qit

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/meerevm/eth"
	mconsensus "github.com/Qitmeer/qng/meerevm/qit/consensus"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethclient"
)

type QitService struct {
	service.Service
	cfg   *config.Config
	cons  model.Consensus
	chain *eth.ETHChain
}

func (q *QitService) Start() error {
	if err := q.Service.Start(); err != nil {
		return err
	}
	log.Info("Start QitService")

	ecfg, args, flags, err := MakeParams(q.cfg)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	chain, err := eth.NewETHChain(ecfg, args, flags)
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
		chain.Ether().Engine().(*mconsensus.Qit).Authorize(eb, wallet.SignData)
		log.Info(fmt.Sprintf("QitSubnet Authorize:%s", eb))
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
		log.Debug(fmt.Sprintf("QitSubnet block chain current block number:%d", blockNum))
	}

	cbh := q.chain.Ether().BlockChain().CurrentBlock()
	if cbh != nil {
		log.Debug(fmt.Sprintf("QitSubnet block chain current block:number=%d hash=%s", cbh.Number.Uint64(), cbh.Hash().String()))
	}

	//
	state, err := q.chain.Ether().BlockChain().State()
	if err != nil {
		return nil
	}
	//
	for addr := range q.chain.Config().Eth.Genesis.Alloc {
		log.Debug(fmt.Sprintf("QitSubnet Alloc address:%v balance:%v", addr.String(), state.GetBalance(addr)))
	}
	return nil
}

func (q *QitService) Stop() error {
	if err := q.Service.Stop(); err != nil {
		return err
	}
	return q.chain.Stop()
}

func (q *QitService) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicQitServiceAPI(q),
			Public:    true,
		},
	}
}

func New(cfg *config.Config, cons model.Consensus) (*QitService, error) {
	a := QitService{
		cfg:  cfg,
		cons: cons,
	}
	return &a, nil
}
