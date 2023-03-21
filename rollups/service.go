package rollups

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rollups/node"
	"github.com/Qitmeer/qng/rollups/node/chaincfg"
	"github.com/Qitmeer/qng/rollups/node/flags"
	"github.com/Qitmeer/qng/rollups/node/heartbeat"
	"github.com/Qitmeer/qng/rollups/node/metrics"
	rn "github.com/Qitmeer/qng/rollups/node/node"
	oplog "github.com/Qitmeer/qng/rollups/service/log"
	oppprof "github.com/Qitmeer/qng/rollups/service/pprof"
	"github.com/Qitmeer/qng/version"
	"github.com/urfave/cli/v2"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
)

const (
	defaultDataDirName = "rollup"
)

func Cmds() *cli.Command {
	return &cli.Command{
		Name:        "rollup",
		Aliases:     []string{"r"},
		Category:    "rollup",
		Usage:       "QNG Rollup Node",
		Description: "The QNG Rollup Node derives L2 block inputs from L1 data and drives an external L2 Execution Engine to build a L2 chain",
		Flags: flags.Flags,
		Action: func(ctx *cli.Context) error {
			cfg := config.Cfg
			defer func() {
				if log.LogWrite() != nil {
					log.LogWrite().Close()
				}
			}()
			interrupt := system.InterruptListener()
			dataPath:=path.Join(cfg.DataDir,defaultDataDirName)
			log.Info("System info", "Rollups Version", version.String(), "Go version", runtime.Version())
			log.Info("System info", "Home dir", dataPath)
			if cfg.NoFileLogging {
				log.Info("File logging disabled")
			}
			showLogo(cfg)
			rn,err:=New(cfg,ctx)
			if err != nil {
				return err
			}
			defer rn.Stop()
			err=rn.Start(interrupt)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

type RollupService struct {
	service.Service
	cfg   *config.Config
	appctx *cli.Context
}

func (q *RollupService) Start(interrupt <-chan struct{}) error {
	if err := q.Service.Start(); err != nil {
		return err
	}
	log.Info("Start RollupService")
	//
	ctx:=q.appctx
	logCfg := oplog.ReadCLIConfig(ctx)
	if err := logCfg.Check(); err != nil {
		log.Error("Unable to create the log config", "error", err)
		return err
	}
	log := oplog.NewLogger(logCfg)
	m := metrics.NewMetrics("default")

	cfg, err := node.NewConfig(ctx, log)
	if err != nil {
		log.Error("Unable to create the rollup node config", "error", err)
		return err
	}
	snapshotLog, err := node.NewSnapshotLogger(ctx)
	if err != nil {
		log.Error("Unable to create snapshot root logger", "error", err)
		return err
	}

	// Only pretty-print the banner if it is a terminal log. Other log it as key-value pairs.
	if logCfg.Format == "terminal" {
		log.Info("rollup config:\n" + cfg.Rollup.Description(chaincfg.L2ChainIDToNetworkName))
	} else {
		cfg.Rollup.LogDescription(log, chaincfg.L2ChainIDToNetworkName)
	}

	n, err := rn.New(context.Background(), cfg, log, snapshotLog, version.String(), m)
	if err != nil {
		log.Error("Unable to create the rollup node", "error", err)
		return err
	}
	log.Info("Starting rollup node", "version", version.String())

	if err := n.Start(context.Background()); err != nil {
		log.Error("Unable to start rollup node", "error", err)
		return err
	}
	defer n.Close()

	m.RecordInfo(version.String())
	m.RecordUp()
	log.Info("Rollup node started")

	if cfg.Heartbeat.Enabled {
		var peerID string
		if cfg.P2P.Disabled() {
			peerID = "disabled"
		} else {
			peerID = n.P2P().Host().ID().String()
		}

		beatCtx, beatCtxCancel := context.WithCancel(context.Background())
		payload := &heartbeat.Payload{
			Version: fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch),
			Meta:    version.Build,
			Moniker: cfg.Heartbeat.Moniker,
			PeerID:  peerID,
			ChainID: cfg.Rollup.L2ChainID.Uint64(),
		}
		go func() {
			if err := heartbeat.Beat(beatCtx, log, cfg.Heartbeat.URL, payload); err != nil {
				log.Error("heartbeat goroutine crashed", "err", err)
			}
		}()
		defer beatCtxCancel()
	}

	if cfg.Pprof.Enabled {
		pprofCtx, pprofCancel := context.WithCancel(context.Background())
		go func() {
			log.Info("pprof server started", "addr", net.JoinHostPort(cfg.Pprof.ListenAddr, strconv.Itoa(cfg.Pprof.ListenPort)))
			if err := oppprof.ListenAndServe(pprofCtx, cfg.Pprof.ListenAddr, cfg.Pprof.ListenPort); err != nil {
				log.Error("error starting pprof", "err", err)
			}
		}()
		defer pprofCancel()
	}
	// Wait until the interrupt signal is received from an OS signal or
	// shutdown is requested through one of the subsystems such as the RPC
	// server.
	<-interrupt
	return nil
}

func (q *RollupService) Stop() error {
	if err := q.Service.Stop(); err != nil {
		return err
	}
	log.Info("Stop RollupService")
	return nil
}

func New(cfg *config.Config,ctx *cli.Context) (*RollupService, error) {
	a := RollupService{
		cfg:  cfg,
		appctx: ctx,
	}
	return &a, nil
}

func showLogo(cfg *config.Config) {
	logo := `
 ██████╗ ███╗   ██╗ ██████╗     ██████╗  ██████╗ ██╗     ██╗     ██╗   ██╗██████╗ 
██╔═══██╗████╗  ██║██╔════╝     ██╔══██╗██╔═══██╗██║     ██║     ██║   ██║██╔══██╗
██║   ██║██╔██╗ ██║██║  ███╗    ██████╔╝██║   ██║██║     ██║     ██║   ██║██████╔╝
██║▄▄ ██║██║╚██╗██║██║   ██║    ██╔══██╗██║   ██║██║     ██║     ██║   ██║██╔═══╝ 
╚██████╔╝██║ ╚████║╚██████╔╝    ██║  ██║╚██████╔╝███████╗███████╗╚██████╔╝██║     
 ╚══▀▀═╝ ╚═╝  ╚═══╝ ╚═════╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚══════╝ ╚═════╝ ╚═╝     
                                                                                  
Rollup: %s Port: %d PID : %d Network : %s https://github.com/Qitmeer/qng

`
	fmt.Printf(logo, version.String(), cfg.P2PTCPPort, os.Getpid(), params.ActiveNetParams.Name)
}