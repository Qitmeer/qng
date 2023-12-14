package simulator

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/core/blockchain"
	_ "github.com/Qitmeer/qng/database/legacydb/ffldb"
	"github.com/Qitmeer/qng/log"
	_ "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/node"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/services/miner"
	"github.com/Qitmeer/qng/version"
	"os"
	"runtime"
)

func DefaultConfig() *config.Config {
	cfg := common.DefaultConfig(os.TempDir())
	cfg.DataDir = ""
	cfg.DevNextGDB = true
	cfg.NoFileLogging = true
	cfg.PrivNet = true
	cfg.DisableRPC = true
	cfg.DisableListen = true
	cfg.NoDiscovery = true
	cfg.Miner = true
	return cfg
}

var mockNodeGlobalID uint

type MockNode struct {
	id          uint
	n           *node.Node
	wallet      *testWallet
	overrideCfg func(cfg *config.Config) error
	//
	publicMinerAPI      *miner.PublicMinerAPI
	privateMinerAPI     *miner.PrivateMinerAPI
	publicBlockAPI      *blockchain.PublicBlockAPI
	publicBlockChainAPI *node.PublicBlockChainAPI
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

	return nil
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

func StartMockNode(overrideCfg func(cfg *config.Config) error) (*MockNode, error) {
	mn := &MockNode{id: mockNodeGlobalID}
	cfg := DefaultConfig()
	if overrideCfg != nil {
		err := overrideCfg(cfg)
		if err != nil {
			return nil, err
		}
	}

	mockNodeGlobalID++
	err := mn.Start(cfg)
	if err != nil {
		return nil, err
	}
	wallet, err := newTestWallet(uint32(mn.id))
	if err != nil {
		return nil, err
	}
	mn.wallet = wallet
	if len(mn.n.Config.MiningAddrs) <= 0 {
		mn.n.Config.SetMiningAddrs(wallet.miningAddr())
	}
	return mn, nil
}
