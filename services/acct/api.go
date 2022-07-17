package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerevm/evm"
)

// PublicEthereumAPI provides an API to access Ethereum full node-related
// information.
type PublicAccountManagerAPI struct {
	a *AccountManager
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for full nodes.
func NewPublicAccountManagerAPI(a *AccountManager) *PublicAccountManagerAPI {
	return &PublicAccountManagerAPI{a}
}

func (api *PublicAccountManagerAPI) GetBalance(address string, coinID types.CoinID) (interface{}, error) {
	if coinID == types.MEERID {
		return api.a.GetBalance(address)
	} else if coinID == types.ETHID {
		cv, err := api.a.chain.VMService.GetVM(evm.MeerEVMID)
		if err != nil {
			return nil, err
		}
		return cv.GetBalance(address)
	}
	return nil, fmt.Errorf("Not support %v", coinID)
}
