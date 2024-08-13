package wallet

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/event"
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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type WalletManager struct {
	service.Service
	qks       *QngKeyStore
	am        *acct.AccountManager
	tm        *tx.TxManager
	cfg       *config.Config
	acc       *accounts.Manager
	events    *event.Feed
	autoClose chan struct{}
	accts     map[common.Address]*Account
}

func (wm *WalletManager) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.WalletNameSpace,
			Service:   NewPrivateWalletAPI(wm),
			Public:    false,
		},
		{
			NameSpace: cmds.WalletNameSpace,
			Service:   NewPublicWalletAPI(wm),
			Public:    true,
		},
	}
}

func New(cfg *config.Config, evm *meer.MeerChain, _am *acct.AccountManager, _tm *tx.TxManager, _events *event.Feed) (*WalletManager, error) {
	conf := evm.ETHChain().Config().Node
	keydir := evm.ETHChain().Node().KeyStoreDir()

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
		events:    _events,
		accts:     map[common.Address]*Account{},
	}
	a.qks = NewQngKeyStore(ks)
	return &a, nil
}

func (wm *WalletManager) Load() error {
	log.Info("Wallet Load Address Start")
	if len(wm.qks.Accounts()) < 1 {
		return fmt.Errorf("not have any wallet,please create one\n ./qng --testnet -A=./ account import")
	}
	a, err := utils.MakeAddress(wm.qks.KeyStore, "0")
	if err != nil {
		return err
	}
	_, key, err := wm.qks.getDecryptedKey(a, wm.cfg.WalletPass)
	if err != nil {
		return err
	}

	wm.qks.mu.Lock()
	defer wm.qks.mu.Unlock()
	addrs, err := GetQngAddrsFromPrivateKey(hex.EncodeToString(key.PrivateKey.D.Bytes()))
	if err != nil {
		return err
	}
	if wm.GetAccount(a.Address) == nil {
		wm.AddAccount(&a, addrs, 0)
	}
	for _, addr := range addrs {
		log.Info("Wallet Load Address", "addr", addr.String())
		_ = wm.am.AddAddress(addr.String())

		u, found := wm.qks.unlocked[addr.String()]
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
		wm.qks.unlocked[addr.String()] = u
		log.Info("Wallet Load Address End", "addr", addr.String())
	}
	log.Info("Wallet Load Address End")
	return nil
}

func (wm *WalletManager) Start() error {
	log.Info("WalletManager start")
	if wm.cfg.AutoCollectEvm {
		err := wm.Load()
		if err != nil {
			return err
		}
	}
	if wm.cfg.AutoCollectEvm {
		go wm.CollectUtxoToEvm()
	}
	if err := wm.Service.Start(); err != nil {
		return err
	}
	return nil
}

func (wm *WalletManager) Stop() error {
	log.Info("WalletManager stop")
	if wm.cfg.AutoCollectEvm {
		wm.autoClose <- struct{}{}
	}
	if err := wm.Service.Stop(); err != nil {
		return err
	}

	return nil
}

func (wm *WalletManager) GetAccount(addr common.Address) *Account {
	if len(wm.accts) <= 0 {
		return nil
	}
	a, ok := wm.accts[addr]
	if !ok {
		return nil
	}
	return a
}

func (wm *WalletManager) GetAccountByIdx(idx int) *Account {
	if len(wm.accts) <= 0 ||
		idx >= len(wm.accts) {
		return nil
	}
	for _, v := range wm.accts {
		if v.Index == idx {
			return v
		}
	}
	return nil
}

func (wm *WalletManager) AddAccount(act *accounts.Account, addrs []types.Address, idx int) *Account {
	ac := &Account{EvmAcct: act, UtxoAccts: addrs, Index: idx}
	wm.accts[act.Address] = ac
	return ac
}

func (wm *WalletManager) ImportRawKey(privkey string, password string) (*Account, error) {
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return nil, err
	}
	index := len(wm.qks.KeyStore.Accounts())
	act, err := wm.qks.KeyStore.ImportECDSA(key, password)
	if err != nil {
		return nil, err
	}
	ac := wm.GetAccount(act.Address)
	if ac != nil {
		return ac, nil
	}
	addrs, err := GetQngAddrsFromPrivateKey(privkey)
	if err != nil {
		return nil, err
	}
	return wm.AddAccount(&act, addrs, index), err
}
