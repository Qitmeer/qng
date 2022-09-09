// Copyright (c) 2017-2018 The qitmeer developers

package address

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
)

// IsForNetwork returns whether or not the address is associated with the
// passed network.
func IsForNetwork(addr types.Address, p *params.Params) bool {
	return addr.IsForNetwork(p.Net)
}

func IsForCurNetwork(addr string) bool {
	add, err := DecodeAddress(addr)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return add.IsForNetwork(params.ActiveNetParams.Params.Net)
}
