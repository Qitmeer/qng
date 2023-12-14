package simulator

import (
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	_ "github.com/Qitmeer/qng/database/legacydb/ffldb"
	"github.com/Qitmeer/qng/log"
	_ "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/node"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/common"
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

	cfg.DebugPrintOrigins = true
	cfg.DebugLevel = "trace"
	return cfg
}

func StartMockNode() (*node.Node, error) {
	cfg := DefaultConfig()
	err := common.SetupConfig(cfg)
	if err != nil {
		return nil, err
	}

	defer func() {
		if log.LogWrite() != nil {
			log.LogWrite().Close()
		}
	}()
	interrupt := system.InterruptListener()

	// Show version and home dir at startup.
	log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version())
	log.Info("System info", "Home dir", cfg.HomeDir)

	if cfg.NoFileLogging {
		log.Info("File logging disabled")
	}

	// Create node and start it.
	n, err := node.NewNode(cfg, params.ActiveNetParams.Params, interrupt)
	if err != nil {
		log.Error("Unable to start server", "listeners", cfg.Listener, "error", err)
		return nil, err
	}
	err = n.RegisterService()
	if err != nil {
		return nil, err
	}
	err = n.Start()
	if err != nil {
		log.Error("Uable to start server", "error", err)
		n.Stop()
		return nil, err
	}
	return n, nil
}
