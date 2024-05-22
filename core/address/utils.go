// Copyright (c) 2017-2018 The qitmeer developers

package address

import (
	"github.com/Qitmeer/qng/common/encode/bech32"
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

// TstAddressTaproot creates an AddressTaproot, initiating the fields as given.
func TstAddressTaproot(version byte, program [32]byte,
	hrp string) *AddressTaproot {

	return &AddressTaproot{
		AddressSegWit{
			hrp:            hrp,
			witnessVersion: version,
			witnessProgram: program[:],
		},
	}
}

// TstAddressTaprootSAddr returns the expected witness program bytes for a
// bech32m encoded P2TR bitcoin address.
func TstAddressTaprootSAddr(addr string) []byte {
	_, data, err := bech32.Decode(addr)
	if err != nil {
		return []byte{}
	}

	// First byte is version, rest is base 32 encoded data.
	data, err = bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return []byte{}
	}
	return data
}
