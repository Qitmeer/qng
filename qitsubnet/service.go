package qitsubnet

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
)

type QitSubnet struct {
	service.Service
	cfg *config.Config
	cons model.Consensus
}

func (q *QitSubnet) Start() error {
	if err := q.Service.Start(); err != nil {
		return err
	}

	return nil
}

func (q *QitSubnet) Stop() error {
	if err := q.Service.Stop(); err != nil {
		return err
	}

	return nil
}

func (q *QitSubnet) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicQitSubnetAPI(q),
			Public:    true,
		},
	}
}

func New(cfg *config.Config,cons model.Consensus) (*QitSubnet, error) {
	a := QitSubnet{
		cfg:      cfg,
		cons: cons,
	}
	return &a, nil
}
