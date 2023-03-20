package wallet

import (
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/acct"
	"github.com/Qitmeer/qng/services/tx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/node"
	"strconv"
)

type WalletManager struct {
	service.Service
	qks *QngKeyStore
	am  *acct.AccountManager
	tm  *tx.TxManager
	cfg *config.Config
	acc *accounts.Manager
}

// PublicWalletManagerAPI provides an API to access Qng wallet function
// information.
type PublicWalletManagerAPI struct {
	a *WalletManager
}

// PrivateWalletManagerAPI provides an API to access Qng wallet function
// information.
type PrivateWalletManagerAPI struct {
	a *WalletManager
}

func NewPrivateWalletAPI(m *WalletManager) *PrivateWalletManagerAPI {
	pmAPI := &PrivateWalletManagerAPI{m}
	return pmAPI
}
func (a *WalletManager) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.WalletNameSpace,
			Service:   NewPrivateWalletAPI(a),
			Public:    false,
		},
	}
}

func New(cfg *config.Config, conf node.Config, _am *acct.AccountManager, _tm *tx.TxManager) (*WalletManager, error) {
	keydir, err := conf.KeyDirConfig()
	if err != nil {
		return nil, err
	}
	n, p := keystore.StandardScryptN, keystore.StandardScryptP
	if conf.UseLightweightKDF {
		n, p = keystore.LightScryptN, keystore.LightScryptP
	}
	ks := keystore.NewKeyStore(keydir, n, p)
	a := WalletManager{
		cfg: cfg,
		am:  _am,
		tm:  _tm,
		acc: accounts.NewManager(&accounts.Config{InsecureUnlockAllowed: conf.InsecureUnlockAllowed},
			ks),
	}
	a.qks = NewQngKeyStore(ks)
	return &a, nil
}

func (acc *WalletManager) MakeAddress(ks *QngKeyStore, account string) (accounts.Account, error) {
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	log.Warn("-------------------------------------------------------------------")
	log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	log.Warn("This functionality is deprecated and will be removed in the future!")
	log.Warn("Please use explicit addresses! (can search via `geth account list`)")
	log.Warn("-------------------------------------------------------------------")

	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}

func (a *WalletManager) Start() error {
	log.Info("WalletManager start")
	if err := a.Service.Start(); err != nil {
		return err
	}
	return nil
}

func (a *WalletManager) Stop() error {
	log.Info("WalletManager stop")
	if err := a.Service.Stop(); err != nil {
		return err
	}
	return nil
}
