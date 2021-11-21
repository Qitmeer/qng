package vm

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/meerevm/evm"
	qconfig "github.com/Qitmeer/qitmeer/config"
	"github.com/Qitmeer/qitmeer/consensus"
	"github.com/Qitmeer/qng/config"
	qconsensus "github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/core/address"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/blockchain/opreturn"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/engine/txscript"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
)

type Factory interface {
	New() (consensus.ChainVM, error)
	GetVM() consensus.ChainVM
	Context() *consensus.Context
}

type Service struct {
	service.Service

	events *event.Feed

	vms map[string]consensus.ChainVM

	cfg *config.Config
}

func (s *Service) Start() error {
	log.Info("Starting Virtual Machines Service")
	if err := s.Service.Start(); err != nil {
		return err
	}
	for _, vm := range s.vms {
		err := vm.Initialize(s.GetVMContext())
		if err != nil {
			return err
		}
		err = vm.Bootstrapping()
		if err != nil {
			return err
		}
		err = vm.Bootstrapped()
		if err != nil {
			return err
		}
	}
	s.subscribe()
	return nil
}

func (s *Service) Stop() error {
	log.Info("Stopping Virtual Machines Service")
	if err := s.Service.Stop(); err != nil {
		return err
	}
	for _, vm := range s.vms {
		vm.Shutdown()
	}
	s.vms = map[string]consensus.ChainVM{}
	return nil
}

func (s *Service) GetVM(id string) (consensus.ChainVM, error) {
	f, ok := s.vms[id]
	if !ok {
		return nil, fmt.Errorf("No VM:%s", id)
	}
	return f, nil
}

func (s *Service) HasVM(id string) bool {
	f, err := s.GetVM(id)
	return err == nil && f != nil
}

func (s *Service) Register(cvm consensus.ChainVM) error {
	if s.HasVM(cvm.GetID()) {
		return fmt.Errorf(fmt.Sprintf("Already exists:%s", cvm.GetID()))
	}

	s.vms[cvm.GetID()] = cvm

	log.Debug(fmt.Sprintf("Register vm %s", cvm.GetID()))
	return nil
}

func (s *Service) Versions() (map[string]string, error) {
	vers := map[string]string{}
	for _, vm := range s.vms {
		vers[vm.GetID()] = vm.Version()
	}
	return vers, nil
}

func (s *Service) registerVMs() error {

	err := s.Register(evm.New())

	return err
}

func (s *Service) GetVMContext() consensus.Context {
	return &qconsensus.Context{
		Context: s.Context(),
		Cfg: &qconfig.Config{
			DataDir:           s.cfg.DataDir,
			DebugLevel:        s.cfg.DebugLevel,
			DebugPrintOrigins: s.cfg.DebugPrintOrigins,
		},
	}
}

func (s *Service) subscribe() {
	ch := make(chan *event.Event)
	sub := s.events.Subscribe(ch)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case ev := <-ch:
				if ev.Data != nil {
					switch value := ev.Data.(type) {
					case *blockchain.Notification:
						s.handleNotifyMsg(value)
					}
				}
				if ev.Ack != nil {
					ev.Ack <- struct{}{}
				}
			}

			if s.IsShutdown() {
				log.Info("Close Miner Event Subscribe")
				return
			}
		}
	}()
}

func (s *Service) handleNotifyMsg(notification *blockchain.Notification) {
	switch notification.Type {
	case blockchain.BlockAccepted:
		ban, ok := notification.Data.(*blockchain.BlockAcceptedNotifyData)
		if !ok {
			return
		}
		vm, err := s.GetVM(evm.MeerEVMID)
		if err == nil {
			txs := []consensus.Tx{}
			for _, tx := range ban.Block.Transactions() {
				if types.IsCrossChainExportTx(tx.Tx) {
					ctx := &qconsensus.Tx{Type: consensus.TxTypeCrossChainExport}
					_, pksAddrs, _, err := txscript.ExtractPkScriptAddrs(tx.Tx.TxOut[0].PkScript, params.ActiveNetParams.Params)
					if err != nil {
						log.Error(err.Error())
						return
					}

					if len(pksAddrs) > 0 {
						secpPksAddr, ok := pksAddrs[0].(*address.SecpPubKeyAddress)
						if !ok {
							log.Error(fmt.Sprintf("Not SecpPubKeyAddress:%s", pksAddrs[0].String()))
							return
						}
						ctx.To = hex.EncodeToString(secpPksAddr.PubKey().SerializeUncompressed())
						ctx.Value = uint64(tx.Tx.TxOut[0].Amount.Value)
						txs = append(txs, ctx)
					}

				} else {
					for _, out := range tx.Tx.TxOut {
						if !opreturn.IsMeerEVM(out.PkScript) {
							continue
						}
						me, err := opreturn.NewOPReturnFrom(out.PkScript)
						if err != nil {
							log.Error(err.Error())
							continue
						}
						ctx := &qconsensus.Tx{Data: []byte(me.(*opreturn.MeerEVM).GetHex())}
						txs = append(txs, ctx)
					}
				}
			}
			if len(txs) <= 0 {
				return
			}
			_, err := vm.BuildBlock(txs)
			if err != nil {
				log.Warn(err.Error())
			}
		}
	}
}

func NewService(cfg *config.Config, events *event.Feed) (*Service, error) {
	ser := Service{
		events: events,
		vms:    make(map[string]consensus.ChainVM),
		cfg:    cfg,
	}
	if err := ser.registerVMs(); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return &ser, nil
}
