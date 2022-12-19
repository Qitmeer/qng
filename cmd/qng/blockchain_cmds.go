package main

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/meerevm/chain"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/version"
	"github.com/Qitmeer/qng/vm"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
)

func blockchainCmd() *cli.Command {
	var (
		outputPath string
		endPoint   string
		byID       bool
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
					db, err := common.LoadBlockDB(cfg)
					if err != nil {
						log.Error("load block database", "error", err)
						return err
					}
					defer func() {
						err = db.Close()
						if err != nil {
							log.Error(err.Error())
						}
					}()
					edbPath := path.Join(cfg.DataDir, chain.ClientIdentifier)
					err = os.RemoveAll(edbPath)
					if err != nil {
						log.Error(err.Error())
					}
					//
					cfg.InvalidTxIndex = false
					cfg.VMBlockIndex = false
					cfg.AddrIndex = false
					cons := consensus.New(cfg, db, interrupt, make(chan struct{}))
					err = cons.Init()
					if err != nil {
						log.Error(err.Error())
						return err
					}
					err = cons.VMService().(*vm.Service).Start()
					if err != nil {
						return err
					}
					defer func() {
						err = cons.VMService().(*vm.Service).Stop()
						if err != nil {
							log.Error(err.Error())
						}
					}()
					if len(outputPath) <= 0 {
						outputPath = cfg.HomeDir
					}
					return export(cons, outputPath, endPoint, byID)
				},
			},
		},
	}
}

func export(consensus model.Consensus, outputPath string, end string, byID bool) error {
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
	bhs := []*hash.Hash{}
	var i uint
	for i = uint(1); i <= endNum; i++ {
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
	logLvl := log.Glogger().GetVerbosity()
	bar := progressbar.Default(int64(endNum-1), "Export:")
	log.Glogger().Verbosity(log.LvlCrit)

	var maxNum [4]byte
	dbnamespace.ByteOrder.PutUint32(maxNum[:], uint32(len(bhs)))
	_, err = outFile.Write(maxNum[:])
	if err != nil {
		return err
	}
	for _, blockHash := range bhs {
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
