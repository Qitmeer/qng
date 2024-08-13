package wallet

import (
	"fmt"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/services/address"
	"github.com/ethereum/go-ethereum/accounts"
)

func GetQngAddrsFromPrivateKey(privateKeyStr string) ([]types.Address, error) {
	_, pkAddr, _, err := address.NewAddresses(privateKeyStr)
	if err != nil {
		return nil, err
	}
	addrs := make([]types.Address, 0)
	addrs = append(addrs, pkAddr)
	addrs = append(addrs, pkAddr.PKHAddress())
	return addrs, nil
}

type Account struct {
	EvmAcct   *accounts.Account
	UtxoAccts []types.Address
	Index     int
}

func (a Account) String() string {
	ret := fmt.Sprintf("%d: %s", a.Index, a.EvmAcct.Address.String())
	for _, v := range a.UtxoAccts {
		ret = fmt.Sprintf("%s %s", ret, v.String())
	}
	return ret
}

func (a Account) PKAddress() types.Address {
	return a.UtxoAccts[0]
}

func (a Account) PKHAddress() types.Address {
	return a.UtxoAccts[1]
}
