package vm

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	qconfig "github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/vm"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerevm/evm"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/vm/consensus"
)

type Factory interface {
	New() (consensus.ChainVM, error)
	GetVM() consensus.ChainVM
	Context() *consensus.Context
}

type Service struct {
	service.Service
	events *event.Feed
	vms    map[string]consensus.ChainVM
	cfg    *config.Config
	apis   []api.API
	ctx    *vm.Context
}

func (s *Service) Start() error {
	log.Info("Starting Virtual Machines Service")
	if err := s.Service.Start(); err != nil {
		return err
	}
	for _, vm := range s.vms {
		vm.RegisterAPIs(s.apis)
		err := vm.Bootstrapping()
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
		_, ok := notification.Data.(*blockchain.BlockAcceptedNotifyData)
		if !ok {
			return
		}

	}
}

func (s *Service) SetLogLevel(level string) {
	for _, vm := range s.vms {
		vm.SetLogLevel(level)
	}
}

func (s *Service) VerifyTx(tx model.Tx) (int64, error) {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0, err
	}
	if tx.GetTxType() == types.TxTypeCrossChainVM {
		return v.VerifyTx(tx)
	}

	if tx.GetTxType() != types.TxTypeCrossChainImport {
		return 0, fmt.Errorf("Not support:%s\n", tx.GetTxType().String())
	}

	itx, ok := tx.(*vm.ImportTx)
	if !ok {
		return 0, fmt.Errorf("Not support tx:%s\n", tx.GetTxType().String())
	}

	pka, err := itx.GetPKAddress()
	if err != nil {
		return 0, err
	}
	ba, err := v.GetBalance(pka.String())
	if err != nil {
		return 0, err
	}
	if ba <= 0 {
		return 0, fmt.Errorf("Balance (%s) is %d\n", pka.String(), ba)
	}
	if ba < itx.Transaction.TxOut[0].Amount.Value {
		return 0, fmt.Errorf("Balance (%s)  %d < output %d", pka.String(), ba, itx.Transaction.TxOut[0].Amount.Value)
	}
	return ba - itx.Transaction.TxOut[0].Amount.Value, nil
}

func (s *Service) VerifyTxSanity(tx model.Tx) error {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return err
	}
	if tx.GetTxType() == types.TxTypeCrossChainVM {
		return v.VerifyTxSanity(tx)
	}
	return nil
}

func (s *Service) AddTxToMempool(tx *types.Transaction, local bool) (int64, error) {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0, err
	}
	return v.AddTxToMempool(tx, local)
}

func (s *Service) RemoveTxFromMempool(tx *types.Transaction) error {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return err
	}
	return v.RemoveTxFromMempool(tx)
}

func (s *Service) GetTxsFromMempool() ([]*types.Transaction, []*hash.Hash, error) {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return nil, nil, err
	}
	return v.GetTxsFromMempool()
}

func (s *Service) GetMempoolSize() int64 {
	v, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0
	}
	return v.GetMempoolSize()
}

func (s *Service) CheckConnectBlock(block *types.SerializedBlock) error {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return err
	}
	b, err := s.normalizeBlock(block, true)
	if err != nil {
		return err
	}

	if len(b.Txs) <= 0 {
		return nil
	}
	return vm.CheckConnectBlock(b)
}

func (s *Service) ConnectBlock(block *types.SerializedBlock) (uint64, error) {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0, err
	}
	b, err := s.normalizeBlock(block, true)
	if err != nil {
		return 0, err
	}

	if len(b.Txs) <= 0 {
		return 0, nil
	}
	return vm.ConnectBlock(b)
}

func (s *Service) DisconnectBlock(block *types.SerializedBlock) (uint64, error) {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0, err
	}
	b, err := s.normalizeBlock(block, false)
	if err != nil {
		return 0, err
	}

	if len(b.Txs) <= 0 {
		return 0, nil
	}
	return vm.DisconnectBlock(b)
}

func (s *Service) normalizeBlock(block *types.SerializedBlock, checkDup bool) (*vm.Block, error) {
	result := &vm.Block{Id: block.Hash(), Txs: []model.Tx{}, Time: block.Block().Header.Timestamp}

	for idx, tx := range block.Transactions() {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate && checkDup {
			continue
		}

		if types.IsCrossChainExportTx(tx.Tx) {
			ctx, err := vm.NewExportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if types.IsCrossChainImportTx(tx.Tx) {
			ctx, err := vm.NewImportTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		} else if types.IsCrossChainVMTx(tx.Tx) {
			ctx, err := vm.NewVMTx(tx.Tx)
			if err != nil {
				return nil, err
			}
			err = ctx.SetCoinbaseTx(block.Transactions()[0].Tx)
			if err != nil {
				return nil, err
			}
			result.Txs = append(result.Txs, ctx)
		}
	}
	return result, nil
}

func (s *Service) RegisterAPIs(apis []api.API) {
	s.apis = append(s.apis, apis...)
}

func (s *Service) ResetTemplate() error {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return err
	}
	return vm.ResetTemplate()
}

func (s *Service) Genesis(txs []*types.Tx) *hash.Hash {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return nil
	}
	hasVM := false
	for idx, tx := range txs {
		if idx == 0 {
			continue
		}
		if tx.IsDuplicate {
			continue
		}
		if types.IsCrossChainExportTx(tx.Tx) {
			hasVM = true
			break
		} else if types.IsCrossChainImportTx(tx.Tx) {
			hasVM = true
			break
		} else if types.IsCrossChainVMTx(tx.Tx) {
			hasVM = true
			break
		}
	}
	if !hasVM {
		return nil
	}
	return vm.Genesis()
}

func (s *Service) GetBlockID(bh *hash.Hash) uint64 {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0
	}
	return vm.GetBlockID(bh)
}

func (s *Service) GetBlockIDByTxHash(txhash *hash.Hash) uint64 {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0
	}
	return vm.GetBlockIDByTxHash(txhash)
}

func (s *Service) SetTxPool(tp model.TxPool) {
	s.ctx.Tp = tp
}

func (s *Service) SetNotify(Notify consensus.Notify) {
	s.ctx.Notify = Notify
}

func (s *Service) GetBalance(addr string) (int64, error) {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return 0, err
	}
	return vm.GetBalance(addr)
}

func (s *Service) GetBlockByNumber(num uint64) (interface{}, error) {
	vm, err := s.GetVM(evm.MeerEVMID)
	if err != nil {
		return nil, err
	}
	return vm.GetBlockByNumber(num)
}

func NewService(cfg *config.Config, events *event.Feed) (*Service, error) {
	ser := Service{
		events: events,
		vms:    make(map[string]consensus.ChainVM),
		cfg:    cfg,
		apis:   []api.API{},
	}
	ser.InitContext()
	ser.ctx = &vm.Context{
		Context: ser.Context(),
		Cfg: &qconfig.Config{
			DataDir:           cfg.DataDir,
			DebugLevel:        cfg.DebugLevel,
			DebugPrintOrigins: cfg.DebugPrintOrigins,
			EVMEnv:            cfg.EVMEnv,
		},
	}
	if err := ser.registerVMs(); err != nil {
		log.Error(err.Error())
		return nil, err
	}

	for _, vm := range ser.vms {
		err := vm.Initialize(ser.ctx)
		if err != nil {
			return nil, err
		}
	}
	return &ser, nil
}
