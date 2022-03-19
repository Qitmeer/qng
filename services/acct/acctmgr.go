package acct

import (
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
)

// account manager communicate with various backends for signing transactions.
type AccountManager struct {
	service.Service
}

func (a AccountManager) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicAccountManagerAPI(&a),
			Public:    true,
		},
	}
}

func New() (*AccountManager, error) {
	a := AccountManager{}
	return &a, nil
}
