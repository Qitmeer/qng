package amana

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/ethereum/go-ethereum/ethclient"
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
	err = q.chain.Start()
	if err != nil {
		return err
	}
	//
	rpcClient := q.chain.Node().Attach()
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

func New(cfg *config.Config, cons model.Consensus) (*AmanaService, error) {
	a := AmanaService{
		cfg:  cfg,
		cons: cons,
	}
	return &a, nil
}
