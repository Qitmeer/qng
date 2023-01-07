package qit

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
)

type QitService struct {
	service.Service
	cfg *config.Config
	cons model.Consensus
}

func (q *QitService) Start() error {
	if err := q.Service.Start(); err != nil {
		return err
	}

	return nil
}

func (q *QitService) Stop() error {
	if err := q.Service.Stop(); err != nil {
		return err
	}

	return nil
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

func New(cfg *config.Config,cons model.Consensus) (*QitService, error) {
	a := QitService{
		cfg:      cfg,
		cons: cons,
	}
	return &a, nil
}
