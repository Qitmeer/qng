package acct

import (
	"fmt"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerevm/meer"
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
		return api.a.chain.MeerChain().(*meer.MeerChain).GetBalance(addr)
	}
	return nil, fmt.Errorf("Not support %v", coinID)
}

func (api *PublicAccountManagerAPI) GetAcctInfo() (interface{}, error) {
	ai := json.AcctInfo{
		Mode:    api.a.cfg.AcctMode,
		Version: api.a.info.version,
		Total:   api.a.info.total,
	}
	api.a.watchLock.RLock()
	ai.Watcher = uint32(len(api.a.watchers))
	api.a.watchLock.RUnlock()
	ai.UtxoWatcher = uint32(api.a.getUtxoWatcherSize())
	if api.a.info.GetAddrTotal() > 0 {
		ai.Addrs = api.a.info.addrs
	}
	return ai, nil
}

func (api *PublicAccountManagerAPI) GetBalanceInfo(addr string, coinID types.CoinID, verbose bool) (interface{}, error) {
	result := BalanceInfoResult{CoinId: coinID.Name()}
	if coinID == types.MEERA {
		bal, err := api.a.GetBalance(addr)
		if err != nil {
			return nil, err
		}
		result.Balance = int64(bal)
		if verbose {
			result.UTXOs, err = api.a.GetUTXOs(addr)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	} else if coinID == types.MEERB {
		ba, err := api.a.chain.MeerChain().(*meer.MeerChain).GetBalance(addr)
		if err != nil {
			return nil, err
		}
		result.Balance = ba
		return result, nil
	}
	return nil, fmt.Errorf("Not support %v", coinID)
}

func (api *PublicAccountManagerAPI) AddBalance(addr string) (interface{}, error) {
	return nil, api.a.AddAddress(addr)
}

func (api *PublicAccountManagerAPI) DelBalance(addr string) (interface{}, error) {
	return nil, api.a.DelAddress(addr)
}
