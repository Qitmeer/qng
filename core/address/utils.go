// Copyright (c) 2017-2018 The qitmeer developers

package address

import (
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
)

// IsForNetwork returns whether or not the address is associated with the
// passed network.
//TODO, other addr type and ec type check
func IsForNetwork(addr types.Address, p *params.Params) bool {
	switch addr := addr.(type) {
	case *PubKeyHashAddress:
		return addr.netID == p.PubKeyHashAddrID
	case *SecpPubKeyAddress:
		return addr.net.Net == p.Net

	}
	return false
}

func IsForCurNetwork(addr string) bool {
	add, err := DecodeAddress(addr)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if !IsForNetwork(add, params.ActiveNetParams.Params) {
		return false
	}
	return true
}
