package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/core/json"
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

func (api *PublicAccountManagerAPI) GetBalance(addr string, coinID types.CoinID) (interface{}, error) {
	if coinID == types.MEERA {
		return api.a.GetBalance(addr)
	} else if coinID == types.MEERB {
		cv, err := api.a.chain.VMService.GetVM(evm.MeerEVMID)
		if err != nil {
			return nil, err
		}
		return cv.GetBalance(addr)
	}
	return nil, fmt.Errorf("Not support %v", coinID)
}

func (api *PublicAccountManagerAPI) GetAcctInfo() (interface{}, error) {
	return json.AcctInfo{
		Mode:    api.a.cfg.AcctMode,
		Version: api.a.info.version,
		Total:   api.a.info.addrTotal,
		Watcher: uint32(len(api.a.watchers)),
	}, nil
}

func (api *PublicAccountManagerAPI) GetBalanceInfo(addr string, coinID types.CoinID) (interface{}, error) {
	result := BalanceInfoResult{CoinId: coinID.Name()}
	if coinID == types.MEERA {
		bal, err := api.a.GetBalance(addr)
		if err != nil {
			return nil, err
		}
		result.Balance = int64(bal)
		result.UTXOs, err = api.a.GetUTXOs(addr)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else if coinID == types.MEERB {
		cv, err := api.a.chain.VMService.GetVM(evm.MeerEVMID)
		if err != nil {
			return nil, err
		}
		ba, err := cv.GetBalance(addr)
		if err != nil {
			return nil, err
		}
		result.Balance = ba
		return result, nil
	}
	return nil, fmt.Errorf("Not support %v", coinID)
}
