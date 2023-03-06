package hotwallet

import (
	"errors"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc/client"
	"github.com/Qitmeer/qng/services/hotwallet/waddrmgs"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"github.com/Qitmeer/qng/services/hotwallet/wtxmgr"
	"sync"
	"time"
)

type Wallet struct {
	cfg *config.Config

	// Data stores
	db      walletdb.DB
	Manager *waddrmgr.Manager
	TxStore *wtxmgr.Store
	tokens  *QitmeerToken

	notificationRpc *client.Client

	// Channels for the manager locker.
	unlockRequests chan unlockRequest
	lockRequests   chan struct{}
	lockState      chan bool

	chainParams *params.Params
	wg          *sync.WaitGroup

	started   bool
	UploadRun bool

	quit   chan struct{}
	quitMu sync.Mutex

	syncAll    bool
	syncLatest bool
	syncOrder  uint32
	toOrder    uint32
	syncQuit   chan struct{}
	syncWg     *sync.WaitGroup
	startScan  bool
	scanEnd    chan struct{}
	orderMutex sync.RWMutex
}
type (
	unlockRequest struct {
		passphrase []byte
		lockAfter  <-chan time.Time // nil prevents the timeout.
		err        chan error
	}
)

var (
	// Namespace bucket keys.
	waddrmgrNamespaceKey = []byte("waddrmgr")
	wtxmgrNamespaceKey   = []byte("wtxmgr")
	tokenmgrNamespaceKey = []byte("tknmgr")
)

// Open loads an already-created wallet from the passed database and namespaces.
func Open(db walletdb.DB, pubPass []byte, _ *waddrmgr.OpenCallbacks,
	params *params.Params, _ uint32, cfg *config.Config) (*Wallet, error) {

	var (
		addrMgr *waddrmgr.Manager
		txMgr   *wtxmgr.Store
		tokens  *QitmeerToken
	)

	// Before attempting to open the wallet, we'll check if there are any
	// database upgrades for us to proceed. We'll also create our references
	// to the address and transaction managers, as they are backed by the
	// database.
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrMgrBucket := tx.ReadWriteBucket(waddrmgrNamespaceKey)
		if addrMgrBucket == nil {
			return errors.New("missing address manager namespace")
		}
		txMgrBucket := tx.ReadWriteBucket(wtxmgrNamespaceKey)
		if txMgrBucket == nil {
			return errors.New("missing transaction manager namespace")
		}
		tokenBucket := tx.ReadWriteBucket(tokenmgrNamespaceKey)
		tokens = NewQitmeerToken(tokenBucket)
		var err error
		addrMgr, err = waddrmgr.Open(addrMgrBucket, pubPass, params)
		if err != nil {
			return err
		}
		txMgr, err = wtxmgr.Open(txMgrBucket, params)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Trace("Opened wallet")

	w := &Wallet{
		wg:             &sync.WaitGroup{},
		syncWg:         &sync.WaitGroup{},
		cfg:            cfg,
		db:             db,
		Manager:        addrMgr,
		TxStore:        txMgr,
		tokens:         tokens,
		unlockRequests: make(chan unlockRequest),
		lockRequests:   make(chan struct{}),
		lockState:      make(chan bool),
		chainParams:    params,
		quit:           make(chan struct{}),
		syncQuit:       make(chan struct{}, 1),
		scanEnd:        make(chan struct{}, 1),
	}

	return w, nil
}

func Create(db walletdb.DB, pubPass, privPass, seed []byte, params *params.Params,
	birthday time.Time) error {

	// If a seed was provided, ensure that it is of valid length. Otherwise,
	// we generate a random seed for the wallet with the recommended seed
	// length.
	return walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrMgrNs, err := tx.CreateTopLevelBucket(waddrmgrNamespaceKey)
		if err != nil {
			return err
		}
		txmgrNs, err := tx.CreateTopLevelBucket(wtxmgrNamespaceKey)
		if err != nil {
			return err
		}
		_, err = tx.CreateTopLevelBucket(tokenmgrNamespaceKey)
		if err != nil {
			return err
		}
		err = waddrmgr.Create(
			addrMgrNs, seed, pubPass, privPass, params, nil,
			birthday,
		)
		if err != nil {
			return err
		}
		return wtxmgr.Create(txmgrNs)
	})
}
