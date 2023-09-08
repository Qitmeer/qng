package wallet

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"

	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/acct"
	"github.com/Qitmeer/qng/services/tx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
)

type WalletManager struct {
	service.Service
	qks           *QngKeyStore
	am            *acct.AccountManager
	tm            *tx.TxManager
	cfg           *config.Config
	acc           *accounts.Manager
	autoCollectOp chan types.AutoCollectUtxo
	autoClose     chan struct{}
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

func New(cfg *config.Config, meer *meer.MeerChain, _am *acct.AccountManager, _tm *tx.TxManager, _autoCollectOp chan types.AutoCollectUtxo) (*WalletManager, error) {
	conf := meer.ETHChain().Config().Node
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
		autoClose: make(chan struct{}),
	}
	a.qks = NewQngKeyStore(ks)
	a.autoCollectOp = _autoCollectOp
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

func (acc *WalletManager) Load() error {
	log.Info("Wallet Load Address Start")
	if len(acc.qks.Accounts()) < 1 {
		return fmt.Errorf("not have any wallet,please create one\n ./qng --testnet -A=./ account import")
	}
	a, err := utils.MakeAddress(acc.qks.KeyStore, "0")
	if err != nil {
		return err
	}
	_, key, err := acc.qks.getDecryptedKey(a, acc.cfg.WalletPass)
	if err != nil {
		return err
	}

	acc.qks.mu.Lock()
	defer acc.qks.mu.Unlock()
	addrs, err := GetQngAddrsFromPrivateKey(hex.EncodeToString(key.PrivateKey.D.Bytes()), acc.am.GetChain().ChainParams())
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		log.Info("Wallet Load Address", "addr", addr.String())
		_ = acc.am.AddAddress(addr.String())

		u, found := acc.qks.unlocked[addr.String()]
		if found {
			if u.abort == nil {
				// The address was unlocked indefinitely, so unlocking
				// it with a timeout would be confusing.
				zeroKey(key.PrivateKey)
				return nil
			}
			// Terminate the expire goroutine and replace it below.
			close(u.abort)
		}
		u = &unlocked{Key: key}
		acc.qks.unlocked[addr.String()] = u
		log.Info("Wallet Load Address End", "addr", addr.String())
	}
	log.Info("Wallet Load Address End")
	return nil
}

func (a *WalletManager) Start() error {
	log.Info("WalletManager start")
	if a.cfg.AutoCollectEvm {
		err := a.Load()
		if err != nil {
			return err
		}
	}
	if a.cfg.AutoCollectEvm {
		go a.CollectUtxoToEvm()
	}
	if err := a.Service.Start(); err != nil {
		return err
	}
	return nil
}

func (a *WalletManager) Stop() error {
	log.Info("WalletManager stop")
	if a.cfg.AutoCollectEvm {
		a.autoClose <- struct{}{}
	}
	if err := a.Service.Stop(); err != nil {
		return err
	}

	return nil
}
