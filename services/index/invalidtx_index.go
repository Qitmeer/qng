package index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/store/invalid_tx_index"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	l "github.com/Qitmeer/qng/log"
	"github.com/schollz/progressbar/v3"
)

const (
	invalidTxIndexName       = "invalid tx index"
	defaultPreallocateCaches = false
	defaultCacheSize         = 10
)

type InvalidTxIndex struct {
	consensus model.Consensus

	invalidtxindexStore model.InvalidTxIndexStore
}

func (idx *InvalidTxIndex) Name() string {
	return invalidTxIndexName
}

func (idx *InvalidTxIndex) Init() error {
	//
	store := idx.invalidtxindexStore
	if store == nil {
		return fmt.Errorf("No invalid tx index store")
	}
	bc := idx.consensus.BlockChain()
	mainOrder := bc.GetMainOrder()
	mainHash := bc.GetBlockHashByOrder(mainOrder)
	if mainHash == nil {
		return fmt.Errorf("No block in order:%d", mainOrder)
	}
	if store.IsEmpty() {
		return idx.caughtUpFrom(0)
	} else {
		tipOrder, tipHash, err := store.Tip(model.NewStagingArea())
		if err != nil {
			return err
		}
		if tipOrder != uint64(mainOrder) || !mainHash.IsEqual(tipHash) {
			if tipOrder < uint64(mainOrder) {
				// It shows that the data is encounter
				bh := bc.GetBlockHashByOrder(uint(tipOrder))
				if bh != nil && bh.IsEqual(tipHash) {
					return idx.caughtUpFrom(uint(tipOrder + 1))
				}
			}
			return fmt.Errorf("vm block index(%s:%d) is out of synchronization(%s:%d) and can only be deleted and rebuilt:index --dropvmblock",
				tipHash, tipOrder, mainHash, mainOrder)
		}
		log.Info(fmt.Sprintf("Current %s tip:%s,%d", idx.Name(), tipHash.String(), tipOrder))
	}
	return nil
}

func (idx *InvalidTxIndex) caughtUpFrom(startOrder uint) error {
	store := idx.invalidtxindexStore
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	bc := idx.consensus.BlockChain()
	mainOrder := bc.GetMainOrder()
	mainHash := bc.GetBlockHashByOrder(mainOrder)
	if startOrder > mainOrder {
		return nil
	}
	if mainOrder > 0 {
		log.Info(fmt.Sprintf("Start caught up %s from (order:%d) to tip(hash:%s,order:%d)", idx.Name(), startOrder, mainHash, mainOrder))
		logLvl := l.Glogger().GetVerbosity()
		bar := progressbar.Default(int64(mainOrder-startOrder), fmt.Sprintf("%s:", idx.Name()))
		l.Glogger().Verbosity(l.LvlCrit)
		for i := uint(startOrder); i <= mainOrder; i++ {
			bar.Add(1)
			if i == 0 {
				continue
			}
			blk := bc.GetBlockByOrder(uint64(i))
			if blk == nil {
				return fmt.Errorf("No DAG block:%d", i)
			}
			if !blk.GetState().GetStatus().KnownInvalid() {
				continue
			}
			block, err := bc.FetchBlockByHash(blk.GetHash())
			if err != nil {
				return err
			}
			err = idx.ConnectBlock(uint64(blk.GetID()), block)
			if err != nil {
				return err
			}
		}
		l.Glogger().Verbosity(logLvl)
	}
	log.Info(fmt.Sprintf("Current %s tip:%s,%d", idx.Name(), mainHash.String(), mainOrder))
	return idx.UpdateMainTip(mainHash, uint64(mainOrder))
}

func (idx *InvalidTxIndex) ConnectBlock(bid uint64, block *types.SerializedBlock) error {
	store := idx.invalidtxindexStore
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.Stage(stagingArea, bid, block)
	return staging.CommitAllChanges(idx.consensus.LegacyDB(), stagingArea)
}

func (idx *InvalidTxIndex) DisconnectBlock(bid uint64, block *types.SerializedBlock) error {
	store := idx.invalidtxindexStore
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.Delete(stagingArea, bid, block)
	return staging.CommitAllChanges(idx.consensus.LegacyDB(), stagingArea)
}

func (idx *InvalidTxIndex) UpdateMainTip(bh *hash.Hash, order uint64) error {
	store := idx.invalidtxindexStore
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.StageTip(stagingArea, bh, order)
	return staging.CommitAllChanges(idx.consensus.LegacyDB(), stagingArea)
}

func (idx *InvalidTxIndex) Get(txid *hash.Hash) (*types.Transaction, error) {
	store := idx.invalidtxindexStore
	if store == nil {
		return nil, fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	return store.Get(stagingArea, txid)
}

func (idx *InvalidTxIndex) GetIdByHash(h *hash.Hash) (*hash.Hash, error) {
	store := idx.invalidtxindexStore
	if store == nil {
		return nil, fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	return store.GetIdByHash(stagingArea, h)
}

func NewInvalidTxIndex(consensus model.Consensus) *InvalidTxIndex {
	log.Info(fmt.Sprintf("%s is enabled", invalidTxIndexName))

	invalidtxindexStore, err := invalid_tx_index.New(consensus.LegacyDB(), consensus.DatabaseContext(), defaultCacheSize, defaultPreallocateCaches)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return &InvalidTxIndex{
		consensus:           consensus,
		invalidtxindexStore: invalidtxindexStore,
	}
}

func DropInvalidTxIndex(db legacydb.DB, interrupt <-chan struct{}) error {
	log.Info("Start drop invalidtx index")
	itiStore, err := invalid_tx_index.New(db, nil, 10, false)
	if err != nil {
		return err
	}
	if itiStore.IsEmpty() {
		return fmt.Errorf("No data needs to be deleted")
	}
	tipOrder, tipHash, err := itiStore.Tip(model.NewStagingArea())
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("All invalidtx index at (%s,%d) will be deleted", tipHash, tipOrder))
	return itiStore.Clean()
}
