package wallet

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/v2/config"
	"github.com/Qitmeer/qng/v2/consensus/model"
	"github.com/Qitmeer/qng/v2/core/event"
	"github.com/Qitmeer/qng/v2/log"
	"github.com/Qitmeer/qng/v2/node/service"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/Qitmeer/qng/v2/rpc/client/cmds"
	"github.com/Qitmeer/qng/v2/services/acct"
	qcommon "github.com/Qitmeer/qng/v2/services/common"
	"github.com/Qitmeer/qng/v2/services/tx"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type WalletManager struct {
	service.Service
	qks       *qcommon.QngKeyStore
	am        *acct.AccountManager
	tm        *tx.TxManager
	cfg       *config.Config
	events    *event.Feed
	autoClose chan struct{}
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

func New(cfg *config.Config, evm model.MeerChain, _am *acct.AccountManager, _tm *tx.TxManager, _events *event.Feed) (*WalletManager, error) {
	a := WalletManager{
		cfg:       cfg,
		am:        _am,
		tm:        _tm,
		events:    _events,
		autoClose: make(chan struct{}),
	}
	if keystores := evm.Node().AccountManager().Backends(keystore.KeyStoreType); len(keystores) > 0 {
		a.qks = keystores[0].(*qcommon.QngKeyStore)
	} else {
		return nil, fmt.Errorf("No keystore backends")
	}
	return &a, nil
}

func (wm *WalletManager) Start() error {
	log.Info("WalletManager start")
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

func (wm *WalletManager) GetAccount(addr common.Address) *qcommon.Account {
	return wm.qks.GetQngAccount(addr.String())
}

func (wm *WalletManager) GetAccountByIdx(idx int) *qcommon.Account {
	return wm.qks.GetQngAccountByIdx(idx)
}

func (wm *WalletManager) ImportRawKey(privkey string, password string) (*accounts.Account, error) {
	if len(password) <= 0 {
		return nil, errors.New("Password can not be empty")
	}
	key, err := crypto.HexToECDSA(privkey)
	if err != nil {
		return nil, err
	}
	act, err := wm.qks.KeyStore.ImportECDSA(key, password)
	if err != nil {
		return nil, err
	}
	return &act, err
}
