package main

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/database/legacychaindb"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/version"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func blockchainCmd() *cli.Command {
	var (
		outputPath string
		endPoint   string
		byID       bool
		inputPath  string
		aidMode    bool
	)
	return &cli.Command{
		Name:        "blockchain",
		Aliases:     []string{"b"},
		Category:    "BlockChain",
		Usage:       "Block Chain",
		Description: "Block Chain",
		Subcommands: []*cli.Command{
			&cli.Command{
				Name:        "export",
				Aliases:     []string{"dump"},
				Usage:       "Write blockchain as a flat file of blocks for use with 'blockchain import', to the specified filename",
				Description: "Export all blocks from database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "path",
						Aliases:     []string{"p"},
						Usage:       "Path to output data",
						Destination: &outputPath,
					},
					&cli.StringFlag{
						Name:        "endpoint",
						Aliases:     []string{"e"},
						Usage:       "End point for output data",
						Destination: &endPoint,
					},
					&cli.BoolFlag{
						Name:        "byid",
						Aliases:     []string{"i"},
						Usage:       "Export by block id",
						Destination: &byID,
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg := config.Cfg
					defer func() {
						if log.LogWrite() != nil {
							log.LogWrite().Close()
						}
					}()
					interrupt := system.InterruptListener()
					log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version())
					log.Info("System info", "Home dir", cfg.HomeDir)
					if cfg.NoFileLogging {
						log.Info("File logging disabled")
					}
					db, err := database.New(cfg, interrupt)
					if err != nil {
						return err
					}
					defer db.Close()
					//
					cfg.InvalidTxIndex = false
					cfg.AddrIndex = false
					cons := consensus.New(cfg, db, interrupt, make(chan struct{}))
					err = cons.Init()
					if err != nil {
						log.Error(err.Error())
						return err
					}
					err = cons.BlockChain().Start()
					if err != nil {
						return err
					}
					defer func() {
						err = cons.BlockChain().Stop()
						if err != nil {
							log.Error(err.Error())
						}
					}()
					if len(outputPath) <= 0 {
						outputPath = cfg.HomeDir
					}
					return exportBlockChain(cons, outputPath, endPoint, byID)
				},
			},
			&cli.Command{
				Name:        "import",
				Aliases:     []string{"i"},
				Usage:       "Import all blocks from database",
				Description: "Import all blocks from database",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "path",
						Aliases:     []string{"p"},
						Usage:       "Path to input data",
						Destination: &inputPath,
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg := config.Cfg
					defer func() {
						if log.LogWrite() != nil {
							log.LogWrite().Close()
						}
					}()
					interrupt := system.InterruptListener()
					log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version())
					log.Info("System info", "Home dir", cfg.HomeDir)
					if cfg.NoFileLogging {
						log.Info("File logging disabled")
					}
					db, err := database.New(cfg, interrupt)
					if err != nil {
						return err
					}
					defer db.Close()
					//
					cfg.InvalidTxIndex = false
					cfg.AddrIndex = false
					cons := consensus.New(cfg, db, interrupt, make(chan struct{}))
					err = cons.Init()
					if err != nil {
						log.Error(err.Error())
						return err
					}
					err = cons.BlockChain().Start()
					if err != nil {
						return err
					}
					defer func() {
						err = cons.BlockChain().Stop()
						if err != nil {
							log.Error(err.Error())
						}
					}()
					if len(inputPath) <= 0 {
						inputPath = cfg.HomeDir
					}
					return importBlockChain(cons, inputPath)
				},
			},
			&cli.Command{
				Name:        "upgrade",
				Aliases:     []string{"u"},
				Usage:       "Upgrade all blocks from database for Qitmeer",
				Description: "Upgrade all blocks from database for Qitmeer",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "path",
						Aliases:     []string{"p"},
						Usage:       "Path to input data",
						Destination: &inputPath,
					},
					&cli.BoolFlag{
						Name:        "aidmode",
						Aliases:     []string{"ai"},
						Usage:       "Export by block id",
						Value:       false,
						Destination: &aidMode,
					},
					&cli.StringFlag{
						Name:        "endpoint",
						Aliases:     []string{"e"},
						Usage:       "End point for output data",
						Destination: &endPoint,
					},
					&cli.BoolFlag{
						Name:        "byid",
						Aliases:     []string{"i"},
						Usage:       "Export by block id",
						Destination: &byID,
					},
				},
				Action: func(ctx *cli.Context) error {
					cfg := config.Cfg
					defer func() {
						if log.LogWrite() != nil {
							log.LogWrite().Close()
						}
					}()
					interrupt := system.InterruptListener()
					log.Info("System info", "QNG Version", version.String(), "Go version", runtime.Version())
					log.Info("System info", "Home dir", cfg.HomeDir)
					if cfg.NoFileLogging {
						log.Info("File logging disabled")
					}
					db, err := database.New(cfg, interrupt)
					if err != nil {
						log.Error("load block database", "error", err)
						return err
					}
					//
					if len(inputPath) <= 0 {
						inputPath = cfg.HomeDir
					}
					return upgradeBlockChain(cfg, db, interrupt, inputPath, endPoint, byID, aidMode)
				},
			},
		},
	}
}

func exportBlockChain(consensus model.Consensus, outputPath string, end string, byID bool) error {
	bc := consensus.BlockChain().(*blockchain.BlockChain)
	mainTip := bc.BlockDAG().GetMainChainTip()
	if mainTip.GetOrder() <= 0 {
		return fmt.Errorf("No blocks in database")
	}
	outFilePath, err := GetIBDFilePath(outputPath)
	if err != nil {
		return err
	}

	outFile, err := os.OpenFile(outFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		outFile.Close()
	}()

	var endPoint meerdag.IBlock
	endNum := uint(0)
	if byID {
		endNum = mainTip.GetID()
	} else {
		endNum = mainTip.GetOrder()
	}

	if len(end) > 0 {
		ephash, err := hash.NewHashFromStr(end)
		if err != nil {
			return err
		}
		endPoint = bc.GetBlock(ephash)
		if endPoint != nil {
			if byID {
				if endNum > endPoint.GetID() {
					endNum = endPoint.GetID()
				}
			} else {
				if endNum > endPoint.GetOrder() {
					endNum = endPoint.GetOrder()
				}
			}

			log.Info(fmt.Sprintf("End point:%s order:%d id:%d", ephash.String(), endPoint.GetOrder(), endPoint.GetID()))
		} else {
			return fmt.Errorf("End point is error")
		}

	}
	logLvl := log.Glogger().GetVerbosity()
	bar := progressbar.Default(int64(endNum-1), "Export:")
	log.Glogger().Verbosity(log.LvlCrit)

	bhs := []*hash.Hash{}
	var i uint
	for i = uint(1); i <= endNum; i++ {
		bar.Add(1)
		var blockHash *hash.Hash
		if byID {
			ib := bc.BlockDAG().GetBlockById(i)
			if ib != nil {
				blockHash = ib.GetHash()
			} else {
				blockHash = nil
			}
		} else {
			blockHash = bc.BlockDAG().GetBlockHashByOrder(i)
		}

		if blockHash == nil {
			if byID {
				log.Trace(fmt.Sprintf("Skip block: Can't find block (%d)!", i))
				continue
			} else {
				return fmt.Errorf(fmt.Sprintf("Can't find block (%d)!", i))
			}
		}
		bhs = append(bhs, blockHash)
	}
	bar.Finish()
	bar.ChangeMax(len(bhs))
	bar.Set(0)
	var maxNum [4]byte
	dbnamespace.ByteOrder.PutUint32(maxNum[:], uint32(len(bhs)))
	_, err = outFile.Write(maxNum[:])
	if err != nil {
		return err
	}
	for _, blockHash := range bhs {
		bar.Add(1)
		block, err := bc.FetchBlockByHash(blockHash)
		if err != nil {
			return err
		}
		bytes, err := block.Bytes()
		if err != nil {
			return err
		}
		ibdb := &IBDBlock{length: uint32(len(bytes)), bytes: bytes}
		err = ibdb.Encode(outFile)
		if err != nil {
			return err
		}
		if bar != nil {
			bar.Add(1)
		}

		/*if endPoint != nil {
			if endPoint.GetHash().IsEqual(blockHash) {
				break
			}
		}*/
	}
	fmt.Println()
	log.Glogger().Verbosity(logLvl)

	log.Info(fmt.Sprintf("Finish export: blocks(%d)    ------>File:%s", len(bhs), outFilePath))
	return nil
}

func importBlockChain(consensus model.Consensus, inputPath string) error {
	bc := consensus.BlockChain().(*blockchain.BlockChain)
	mainTip := bc.BlockDAG().GetMainChainTip()
	if mainTip.GetOrder() > 0 {
		return fmt.Errorf("Your database is not empty, please empty the database.")
	}
	inputFilePath, err := GetIBDFilePath(inputPath)
	if err != nil {
		return err
	}
	blocksBytes, err := util.ReadFile(inputFilePath)
	if err != nil {
		return err
	}
	offset := 0
	maxOrder := dbnamespace.ByteOrder.Uint32(blocksBytes[offset : offset+4])
	offset += 4

	logLvl := log.Glogger().GetVerbosity()
	bar := progressbar.Default(int64(maxOrder-1), "Import:")
	log.Glogger().Verbosity(log.LvlCrit)

	for i := uint32(1); i <= maxOrder; i++ {
		ibdb := &IBDBlock{}
		err := ibdb.Decode(blocksBytes[offset:])
		if err != nil {
			return err
		}
		offset += 4 + int(ibdb.length)

		err = bc.CheckBlockSanity(ibdb.blk, bc.TimeSource(), blockchain.BFFastAdd, params.ActiveNetParams.Params)
		if err != nil {
			return err
		}
		_, err = bc.FastAcceptBlock(ibdb.blk, blockchain.BFFastAdd)
		if err != nil {
			return err
		}
		if bar != nil {
			bar.Add(1)
		}
	}
	fmt.Println()
	log.Glogger().Verbosity(logLvl)

	mainTip = bc.BlockDAG().GetMainChainTip()
	log.Info(fmt.Sprintf("Finish import: blocks(%d)    ------>File:%s", mainTip.GetOrder(), inputFilePath))
	log.Info(fmt.Sprintf("New Info:%s  mainOrder=%d tips=%d", mainTip.GetHash().String(), mainTip.GetOrder(), bc.BlockDAG().GetTips().Size()))
	return nil
}

func upgradeBlockChain(cfg *config.Config, cdb model.DataBase, interrupt <-chan struct{}, inputPath string, end string, byID bool, aidMode bool) error {
	// new block chain
	var err error
	newCfg := *cfg
	newCfg.DataDir, err = util.GetPathByBrother("temp", cfg.DataDir)
	if err != nil {
		return err
	}
	database.Cleanup(&newCfg)
	time.Sleep(time.Second * 2)

	newdb, err := database.New(&newCfg, interrupt)
	if err != nil {
		log.Error("load block database", "error", err)
		return err
	}
	defer func() {
		if newdb != nil {
			newdb.Close()
			time.Sleep(time.Second * 2)
			database.Cleanup(&newCfg)
		}
	}()
	//
	newCfg.InvalidTxIndex = false
	newCfg.AddrIndex = false
	newcons := consensus.New(&newCfg, newdb, interrupt, make(chan struct{}))
	err = newcons.Init()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	err = newcons.BlockChain().Start()
	if err != nil {
		return err
	}
	defer func() {
		if newcons != nil {
			err = newcons.BlockChain().Stop()
			if err != nil {
				log.Error(err.Error())
			}
		}

	}()
	newbc := newcons.BlockChain().(*blockchain.BlockChain)
	err = newbc.Start()
	if err != nil {
		return err
	}
	defer func() {
		if newbc != nil {
			err = newbc.Stop()
			if err != nil {
				log.Error(err.Error())
			}
		}

	}()
	log.Info(fmt.Sprintf("Be ready workspace:%s", newCfg.DataDir))

	processBlock := func(block *types.SerializedBlock, bar *progressbar.ProgressBar) error {
		er := newbc.CheckBlockSanity(block, newbc.TimeSource(), blockchain.BFFastAdd, params.ActiveNetParams.Params)
		if er != nil {
			return er
		}
		_, er = newbc.FastAcceptBlock(block, blockchain.BFFastAdd)
		if er != nil {
			return er
		}
		if bar != nil {
			bar.Add(1)
		}
		return nil
	}

	logLvl := log.Glogger().GetVerbosity()
	//
	db := cdb.(*legacychaindb.LegacyChainDB).DB()
	if aidMode {
		endNum := uint(0)
		serializedData, err := cdb.GetBestChainState()
		if err != nil {
			return err
		}
		if serializedData == nil {
			return nil
		}
		log.Info("Serialized chain state: ", "serializedData", fmt.Sprintf("%x", serializedData))
		state, err := blockchain.DeserializeBestChainState(serializedData)
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("blocks:%d", state.GetTotal()))
		if state.GetTotal() == 0 {
			return fmt.Errorf("No blocks in database")
		}
		endNum = uint(state.GetTotal() - 1)

		if len(end) > 0 {
			endV, err := strconv.Atoi(end)
			if err != nil {
				return err
			}
			if endV > 0 && int64(endV) < int64(endNum) {
				endNum = uint(endV)
			}
		}
		log.Info("Target end point", "block id", endNum)

		bar := progressbar.Default(int64(endNum-1), "Upgrade:")
		log.Glogger().Verbosity(log.LvlCrit)
		eth.InitLog(log.LvlCrit.String(), cfg.DebugPrintOrigins)
		var i uint
		var blockHash *hash.Hash

		for i = uint(1); i <= endNum; i++ {
			bar.Add(1)
			blockHash = nil
			dblock := &meerdag.Block{}
			dblock.SetID(i)
			ib := &meerdag.PhantomBlock{Block: dblock}

			err := meerdag.DBGetDAGBlock(cdb, ib)
			if err != nil {
				if err.(*meerdag.DAGError).IsEmpty() {
					continue
				} else {
					return err
				}
			}
			blockHash = ib.GetHash()
			if err != nil {
				return err
			}

			if blockHash == nil {
				return fmt.Errorf(fmt.Sprintf("Can't find block (%d)!", i))
			}

			block, err := cdb.GetBlock(blockHash)
			if err != nil {
				return err
			}
			//
			err = processBlock(block, bar)
			if err != nil {
				return err
			}

			if system.InterruptRequested(interrupt) {
				break
			}
		}
		fmt.Println()
		log.Glogger().Verbosity(logLvl)
		eth.InitLog(logLvl.String(), cfg.DebugPrintOrigins)
	} else {
		cfg.InvalidTxIndex = false
		cfg.AddrIndex = false
		cons := consensus.New(cfg, cdb, interrupt, make(chan struct{}))
		err := cons.Init()
		if err != nil {
			log.Error(err.Error())
			return err
		}
		err = cons.BlockChain().Start()
		if err != nil {
			return err
		}
		bc := cons.BlockChain().(*blockchain.BlockChain)
		mainTip := bc.BlockDAG().GetMainChainTip()
		if mainTip.GetOrder() <= 0 {
			return fmt.Errorf("No blocks in database")
		}

		var endPoint meerdag.IBlock
		endNum := uint(0)
		if byID {
			endNum = mainTip.GetID()
		} else {
			endNum = mainTip.GetOrder()
		}

		if len(end) > 0 {
			ephash, err := hash.NewHashFromStr(end)
			if err != nil {
				return err
			}
			endPoint = bc.GetBlock(ephash)
			if endPoint != nil {
				if byID {
					if endNum > endPoint.GetID() {
						endNum = endPoint.GetID()
					}
				} else {
					if endNum > endPoint.GetOrder() {
						endNum = endPoint.GetOrder()
					}
				}

				log.Info(fmt.Sprintf("End point:%s order:%d id:%d", ephash.String(), endPoint.GetOrder(), endPoint.GetID()))
			} else {
				return fmt.Errorf("End point is error")
			}

		}

		bar := progressbar.Default(int64(endNum-1), "Export:")
		log.Glogger().Verbosity(log.LvlCrit)
		eth.InitLog(log.LvlCrit.String(), cfg.DebugPrintOrigins)

		var i uint
		var blockHash *hash.Hash

		for i = uint(1); i <= endNum; i++ {
			bar.Add(1)
			if byID {
				ib := bc.BlockDAG().GetBlockById(i)
				if ib != nil {
					blockHash = ib.GetHash()
				} else {
					blockHash = nil
				}
			} else {
				blockHash = bc.BlockDAG().GetBlockHashByOrder(i)
			}

			if blockHash == nil {
				return fmt.Errorf(fmt.Sprintf("Can't find block (%d)!", i))
			}

			block, err := bc.FetchBlockByHash(blockHash)
			if err != nil {
				return err
			}
			err = processBlock(block, bar)
			if err != nil {
				return err
			}
			if system.InterruptRequested(interrupt) {
				break
			}
		}
		fmt.Println()
		log.Glogger().Verbosity(logLvl)
		eth.InitLog(logLvl.String(), cfg.DebugPrintOrigins)

		err = cons.BlockChain().Stop()
		if err != nil {
			log.Error(err.Error())
			return err
		}
	}
	err = newbc.Stop()
	if err != nil {
		log.Error(err.Error())
	}
	total := newbc.BlockDAG().GetBlockTotal()
	newbc = nil
	err = newcons.BlockChain().Stop()
	if err != nil {
		log.Error(err.Error())
	}
	newcons = nil
	newdb.Close()
	newdb = nil
	//
	log.Info(fmt.Sprintf("Gracefully shutting down the last database:%s", cfg.DataDir))
	db.Close()
	err = os.Rename(filepath.Join(cfg.DataDir, "network.key"), filepath.Join(newCfg.DataDir, "network.key"))
	if err != nil {
		log.Info(err.Error())
	}
	time.Sleep(time.Second * 2)
	err = os.RemoveAll(cfg.DataDir)
	if err != nil {
		log.Info(err.Error())
	}
	time.Sleep(time.Second * 2)

	err = os.Rename(newCfg.DataDir, cfg.DataDir)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Finish upgrade: blocks(%d)", total))
	return nil
}

func GetIBDFilePath(path string) (string, error) {
	if len(path) <= 0 {
		return "", fmt.Errorf("Path error")
	}
	if len(path) >= 4 {
		if path[len(path)-4:] == ".ibd" {
			return path, nil
		}
	}
	const defaultFileName = "blocks.ibd"
	return strings.TrimRight(strings.TrimRight(path, "/"), "\\") + "/" + defaultFileName, nil
}

type IBDBlock struct {
	length uint32
	bytes  []byte
	blk    *types.SerializedBlock
}

func (b *IBDBlock) Encode(w io.Writer) error {
	var serializedLen [4]byte
	dbnamespace.ByteOrder.PutUint32(serializedLen[:], b.length)
	_, err := w.Write(serializedLen[:])
	if err != nil {
		return err
	}
	_, err = w.Write(b.bytes)
	return err
}

func (b *IBDBlock) Decode(bytes []byte) error {
	b.length = dbnamespace.ByteOrder.Uint32(bytes[:4])

	block, err := types.NewBlockFromBytes(bytes[4 : b.length+4])
	if err != nil {
		return err
	}
	b.blk = block
	return nil
}
