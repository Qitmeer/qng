package hotwallet

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/hotwallet/waddrmgs"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	walletDbName = "wallet.db"
)

var (
	// ErrLoaded describes the error condition of attempting to load or
	// create a wallet when the loader has already done so.
	ErrLoaded = errors.New("wallet already loaded")

	// ErrNotLoaded describes the error condition of attempting to close a
	// loaded wallet when a wallet has not been loaded.
	ErrNotLoaded = errors.New("wallet is not loaded")

	// ErrExists describes the error condition of attempting to create a new
	// wallet when one exists already.
	ErrExists = errors.New("wallet already exists")
)

type WalletService struct {
	service.Service
	cfg    *config.Config
	BC     *blockchain.BlockChain
	DB     walletdb.DB
	Wallet *Wallet
	mu     sync.Mutex
}

func (w *WalletService) Start() error {
	return nil
}

func (w *WalletService) Stop() error {
	return nil
}

func (w *WalletService) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicWalletServiceAPI(w),
			Public:    true,
		},
	}
}

func New(cfg *config.Config, bc *blockchain.BlockChain) (*WalletService, error) {
	a := WalletService{
		cfg: cfg,
		BC:  bc,
	}
	// create wallet

	return &a, nil
}

// WalletExists returns whether a file exists at the loader's database path.
// This may return an error for unexpected I/O failures.
func (w *WalletService) WalletExists() (bool, error) {
	dbPath := filepath.Join(w.cfg.DataDir, walletDbName)
	return fileExists(dbPath)
}

var errNoConsole = errors.New("db upgrade requires console access for additional input")

func noConsole() ([]byte, error) {
	return nil, errNoConsole
}

// OpenExistingWallet opens the wallet from the loader's wallet database path
// and the public passphrase.  If the loader is being called by a context where
// standard input prompts may be used during wallet upgrades, setting
// canConsolePrompt will enables these prompts.
func (l *WalletService) OpenExistingWallet(pubPassphrase []byte, canConsolePrompt bool) (*Wallet, error) {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.Wallet != nil {
		return nil, ErrLoaded
	}

	// Ensure that the network directory exists.
	if err := checkCreateDir(l.cfg.DataDir); err != nil {
		return nil, err
	}

	// Open the database using the boltdb backend.
	dbPath := filepath.Join(l.cfg.DataDir, walletDbName)
	log.Trace("OpenExistingWallet", "dbPath", dbPath)

	db, err := walletdb.Open("bdb", dbPath)
	if err != nil {
		log.Trace("Failed to open database", "err", err)
		return nil, err
	}
	log.Trace("OpenExistingWallet", "open db", "succ")

	var cbs *waddrmgr.OpenCallbacks
	if !canConsolePrompt {
		cbs = &waddrmgr.OpenCallbacks{
			ObtainSeed:        noConsole,
			ObtainPrivatePass: noConsole,
		}
	}

	log.Trace("OpenExistingWallet", "open", "1")
	w, err := Open(db, pubPassphrase, cbs, l.BC.ChainParams(), 250, l.cfg)
	if err != nil {
		log.Trace("OpenExistingWallet", "open error", err)
		// If opening the wallet fails (e.g. because of wrong
		// passphrase), we must close the backing database to
		// allow future calls to walletdb.Open().
		e := db.Close()
		if e != nil {
			log.Warn("Error closing database: %v", e)
		}
		return nil, err
	}
	log.Trace("OpenExistingWallet", "open ok", true)

	l.onLoaded(w, db)
	return w, nil
}

// onLoaded executes each added callback and prevents loader from loading any
// additional wallets.  Requires mutex to be locked.
func (l *WalletService) onLoaded(w *Wallet, db walletdb.DB) {
	l.Wallet = w
	l.DB = db
}

func (w *WalletService) OpenWallet() (*Wallet, error) {
	b, err := w.WalletExists()
	if err != nil {
		return nil, err
	}
	if b {
		return w.OpenExistingWallet(nil, false)
	} else {
		return w.CreateNewWallet(nil, nil, nil, time.Now())
	}
}

// CreateNewWallet creates a new wallet using the provided public and private
// passphrases.  The seed is optional.  If non-nil, addresses are derived from
// this seed.  If nil, a secure random seed is generated.
func (l *WalletService) CreateNewWallet(pubPassphrase, privPassphrase, seed []byte,
	bday time.Time) (*Wallet, error) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.Wallet != nil {
		return nil, ErrLoaded
	}

	dbPath := filepath.Join(l.cfg.DataDir, walletDbName)
	exists, err := fileExists(dbPath)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrExists
	}

	// Create the wallet database backed by bolt db.
	err = os.MkdirAll(l.cfg.DataDir, 0700)
	if err != nil {
		return nil, err
	}
	db, err := walletdb.Create("bdb", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize the newly created database for the wallet before opening.
	err = Create(
		db, pubPassphrase, privPassphrase, seed, l.BC.ChainParams(), bday,
	)
	if err != nil {
		return nil, err
	}

	// Open the newly-created wallet.
	w, err := Open(db, pubPassphrase, nil, l.BC.ChainParams(), 250, l.cfg)
	if err != nil {
		return nil, err
	}
	//w.Start()

	l.onLoaded(w, db)
	return w, nil
}

func fileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// checkCreateDir checks that the path exists and is a directory.
// If path does not exist, it is created.
func checkCreateDir(path string) error {
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// Attempt data directory creation
			if err = os.MkdirAll(path, 0700); err != nil {
				return fmt.Errorf("cannot create directory: %s", err)
			}
		} else {
			return fmt.Errorf("error checking directory: %s", err)
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("path '%s' is not a directory", path)
		}
	}

	return nil
}
