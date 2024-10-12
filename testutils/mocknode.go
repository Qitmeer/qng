package testutils

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/meerevm/proxy"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"math/rand"
	"os"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types/pow"
	_ "github.com/Qitmeer/qng/database/legacydb/ffldb"
	"github.com/Qitmeer/qng/log"
	_ "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/node"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/acct"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/miner"
	"github.com/Qitmeer/qng/services/tx"
	"github.com/Qitmeer/qng/services/wallet"
	"github.com/Qitmeer/qng/testutils/testprivatekey"
	"github.com/Qitmeer/qng/version"
	"github.com/ethereum/go-ethereum/ethclient"
)

func DefaultConfig(pb *testprivatekey.Builder) (*config.Config, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg := common.DefaultConfig(path.Join(os.TempDir(), fmt.Sprintf("qng_%d_%d", mockNodeGlobalID, r.Uint32())))
	cfg.DataDir = ""
	cfg.DevNextGDB = true
	cfg.NoFileLogging = true
	cfg.PrivNet = true
	cfg.DisableRPC = true
	cfg.DisableListen = true
	cfg.NoDiscovery = true
	cfg.Miner = true
	cfg.SubmitNoSynced = true
	cfg.AcctMode = true
	cfg.EVMEnv = "--nodiscover --v5disc=false --rpc.allow-unprotected-txs"

	params.ActiveNetParams = &params.PrivNetParam
	coinbasePKHex := pb.GetHex(testprivatekey.CoinbaseIdx)
	addrs, err := wallet.GetQngAddrsFromPrivateKey(coinbasePKHex)
	if err != nil {
		return nil, err
	}
	cfg.SetMiningAddrs(addrs[0])
	return cfg, nil
}

var mockNodeGlobalID uint32
var mockNodeLock sync.RWMutex

type MockNode struct {
	id          uint32
	n           *node.Node
	pb          *testprivatekey.Builder
	overrideCfg func(cfg *config.Config) error
	//
	publicMinerAPI          *miner.PublicMinerAPI
	privateMinerAPI         *miner.PrivateMinerAPI
	publicBlockAPI          *blockchain.PublicBlockAPI
	publicBlockChainAPI     *node.PublicBlockChainAPI
	publicTxAPI             *tx.PublicTxAPI
	privateTxAPI            *tx.PrivateTxAPI
	publicAccountManagerAPI *acct.PublicAccountManagerAPI
	privateWalletManagerAPI *wallet.PrivateWalletManagerAPI
	publicWalletManagerAPI  *wallet.PublicWalletManagerAPI
	evmClient               *ethclient.Client
	walletManager           *wallet.WalletManager
	publicMeerChainAPI      *meer.PublicMeerChainAPI
}

func (mn *MockNode) ID() uint {
	return uint(mn.id)
}

func (mn *MockNode) Start(cfg *config.Config) error {
	err := common.SetupConfig(cfg)
	if err != nil {
		return err
	}

	interrupt := system.InterruptListener()

	// Show version and home dir at startup.
	log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version(), "ID", mn.id)
	log.Info("System info", "Home dir", cfg.HomeDir)

	if cfg.NoFileLogging {
		log.Info("File logging disabled")
	}

	// Create node and start it.
	n, err := node.NewNode(cfg, params.ActiveNetParams.Params, interrupt)
	if err != nil {
		log.Error("Unable to start server", "listeners", cfg.Listener, "error", err)
		return err
	}
	mn.n = n
	err = n.RegisterService()
	if err != nil {
		return err
	}
	err = n.Start()
	if err != nil {
		log.Error("Uable to start server", "error", err)
		n.Stop()
		return err
	}

	return mn.setup()
}

func (mn *MockNode) Stop() {
	if log.LogWrite() != nil {
		log.LogWrite().Close()
	}
	if mn.n != nil {
		err := mn.n.Stop()
		if err != nil {
			log.Error(err.Error())
		}
	}
	// remove temp dir
	log.Info("Try remove home dir", "path", mn.n.Config.HomeDir)
	err := os.RemoveAll(mn.n.Config.HomeDir)
	if err != nil {
		log.Error(err.Error())
	}
}

func (mn *MockNode) setup() error {
	mn.walletManager = mn.n.GetQitmeerFull().GetWalletManager()
	// init
	coinbasePKHex := mn.pb.GetHex(testprivatekey.CoinbaseIdx)
	account, err := mn.walletManager.ImportRawKey(coinbasePKHex, testprivatekey.Password)
	if err != nil {
		return err
	}
	err = mn.GetPrivateWalletManagerAPI().Unlock(account.EvmAcct.Address.String(), testprivatekey.Password, time.Hour)
	if err != nil {
		return err
	}

	//
	ethchain := mn.n.GetQitmeerFull().GetBlockChain().MeerChain().(*meer.MeerChain).ETHChain()
	backends := ethchain.Backend().AccountManager().Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		return fmt.Errorf("Failed to unlock accounts, keystore is not available")
	}
	ks := backends[0].(*keystore.KeyStore)
	/*pk, err := crypto.ToECDSA(mn.pb.Get(testprivatekey.CoinbaseIdx))
	if err != nil {
		return err
	}
	acc, err := ks.ImportECDSA(pk, testprivatekey.Password)
	if err != nil {
		return err
	}*/
	err = ks.Unlock(*account.EvmAcct, testprivatekey.Password)
	if err != nil {
		return err
	}
	//

	log.Info("Import default key", "addr", account.String())

	mn.Node().GetQitmeerFull().GetMiner().NoDevelopGap = true
	params.ActiveNetParams.PowConfig.DifficultyMode = pow.DIFFICULTY_MODE_DEVELOP
	return nil
}

func (mn *MockNode) GetPublicMinerAPI() *miner.PublicMinerAPI {
	if mn.publicMinerAPI == nil {
		mn.publicMinerAPI = miner.NewPublicMinerAPI(mn.n.GetQitmeerFull().GetMiner())
	}
	return mn.publicMinerAPI
}

func (mn *MockNode) GetPrivateMinerAPI() *miner.PrivateMinerAPI {
	if mn.privateMinerAPI == nil {
		mn.privateMinerAPI = miner.NewPrivateMinerAPI(mn.n.GetQitmeerFull().GetMiner())
	}
	return mn.privateMinerAPI
}

func (mn *MockNode) GetPublicBlockAPI() *blockchain.PublicBlockAPI {
	if mn.publicBlockAPI == nil {
		mn.publicBlockAPI = blockchain.NewPublicBlockAPI(mn.n.GetQitmeerFull().GetBlockChain())
	}
	return mn.publicBlockAPI
}

func (mn *MockNode) GetPublicBlockChainAPI() *node.PublicBlockChainAPI {
	if mn.publicBlockChainAPI == nil {
		mn.publicBlockChainAPI = node.NewPublicBlockChainAPI(mn.n.GetQitmeerFull())
	}
	return mn.publicBlockChainAPI
}

func (mn *MockNode) GetPublicTxAPI() *tx.PublicTxAPI {
	if mn.publicTxAPI == nil {
		mn.publicTxAPI = tx.NewPublicTxAPI(mn.n.GetQitmeerFull().GetTxManager())
	}
	return mn.publicTxAPI
}

func (mn *MockNode) GetPrivateTxAPI() *tx.PrivateTxAPI {
	if mn.privateTxAPI == nil {
		mn.privateTxAPI = tx.NewPrivateTxAPI(mn.n.GetQitmeerFull().GetTxManager())
	}
	return mn.privateTxAPI
}

func (mn *MockNode) GetPublicAccountManagerAPI() *acct.PublicAccountManagerAPI {
	if mn.publicAccountManagerAPI == nil {
		mn.publicAccountManagerAPI = acct.NewPublicAccountManagerAPI(mn.n.GetQitmeerFull().GetAccountManager())
	}
	return mn.publicAccountManagerAPI
}

func (mn *MockNode) GetPrivateWalletManagerAPI() *wallet.PrivateWalletManagerAPI {
	if mn.privateWalletManagerAPI == nil {
		mn.privateWalletManagerAPI = wallet.NewPrivateWalletAPI(mn.n.GetQitmeerFull().GetWalletManager())
	}
	return mn.privateWalletManagerAPI
}

func (mn *MockNode) GetEvmClient() *ethclient.Client {
	if mn.evmClient == nil {
		mn.evmClient = ethclient.NewClient(mn.n.GetQitmeerFull().GetBlockChain().MeerChain().(*meer.MeerChain).ETHChain().Node().Attach())
	}
	return mn.evmClient
}

func (mn *MockNode) GetPublicWalletManagerAPI() *wallet.PublicWalletManagerAPI {
	if mn.publicWalletManagerAPI == nil {
		mn.publicWalletManagerAPI = wallet.NewPublicWalletAPI(mn.n.GetQitmeerFull().GetWalletManager())
	}
	return mn.publicWalletManagerAPI
}

func (mn *MockNode) GetWalletManager() *wallet.WalletManager {

	return mn.walletManager
}

func (mn *MockNode) GetBuilder() *testprivatekey.Builder {
	return mn.pb
}

func (mn *MockNode) DeterministicDeploymentProxy() *proxy.DeterministicDeploymentProxy {
	return mn.n.GetQitmeerFull().GetBlockChain().MeerChain().(*meer.MeerChain).DeterministicDeploymentProxy()
}

func (mn *MockNode) Node() *node.Node {
	return mn.n
}

func (mn *MockNode) NewAddress() (*wallet.Account, error) {
	// init
	pkb, err := mn.pb.Build()
	if err != nil {
		return nil, err
	}

	account, err := mn.walletManager.ImportRawKey(hex.EncodeToString(pkb), testprivatekey.Password)
	if err != nil {
		return nil, err
	}
	err = mn.GetPrivateWalletManagerAPI().Unlock(account.EvmAcct.Address.String(), testprivatekey.Password, time.Hour)
	if err != nil {
		return nil, err
	}

	return account, nil
}

func (mn *MockNode) GetPriKeyBuilder() *testprivatekey.Builder {
	return mn.pb
}

func (mn *MockNode) HasTx(id *hash.Hash) bool {
	return mn.n.GetQitmeerFull().GetBlockChain().HasTx(id)
}

func (mn *MockNode) GetPublicMeerChainAPI() *meer.PublicMeerChainAPI {
	if mn.publicMeerChainAPI == nil {
		mn.publicMeerChainAPI = meer.NewPublicMeerChainAPI(mn.n.GetQitmeerFull().GetBlockChain().MeerChain().(*meer.MeerChain))
	}
	return mn.publicMeerChainAPI
}

func StartMockNode(overrideCfg func(cfg *config.Config) error) (*MockNode, error) {
	mockNodeLock.Lock()
	defer mockNodeLock.Unlock()

	pb, err := testprivatekey.NewBuilder(0)
	if err != nil {
		return nil, err
	}
	mn := &MockNode{id: mockNodeGlobalID, pb: pb}
	cfg, err := DefaultConfig(pb)
	if err != nil {
		return nil, err
	}
	if overrideCfg != nil {
		err := overrideCfg(cfg)
		if err != nil {
			return nil, err
		}
	}
	err = mn.Start(cfg)
	if err != nil {
		return nil, err
	}
	mockNodeGlobalID++
	return mn, nil
}
