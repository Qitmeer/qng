// Copyright (c) 2017-2018 The qitmeer developers

package vm

import (
	"fmt"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
)

func (s *Service) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicVMAPI(s),
			Public:    true,
		},
	}
}

type PublicVMAPI struct {
	ser *Service
}

func NewPublicVMAPI(s *Service) *PublicVMAPI {
	pmAPI := &PublicVMAPI{ser: s}
	return pmAPI
}

func (api *PublicVMAPI) GetVMsInfo() (interface{}, error) {
	vs, err := api.ser.Versions()
	if err != nil {
		return nil, err
	}
	result := []string{}
	for id, v := range vs {
		result = append(result, fmt.Sprintf("%s:%s", id, v))
	}
	return result, nil
}
