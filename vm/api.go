// Copyright (c) 2017-2018 The qitmeer developers

package vm

import (
	"encoding/json"
	qjson "github.com/Qitmeer/qng-core/core/json"
	"github.com/Qitmeer/qng-core/rpc/api"
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
	result := qjson.OrderedResult{}
	for id, v := range vs {
		strv:=map[string]string{}
		err=json.Unmarshal([]byte(v),&strv)
		if err != nil {
			log.Error(err.Error())
			continue
		}
		result=append(result,qjson.KV{Key:id,Val:strv})
	}
	return result, nil
}
