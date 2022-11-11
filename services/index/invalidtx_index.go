package index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"github.com/schollz/progressbar/v3"
)

const (
	invalidTxIndexName = "invalid tx index"
)

type InvalidTxIndex struct {
	consensus model.Consensus
}

func (idx *InvalidTxIndex) Name() string {
	return invalidTxIndexName
}

func (idx *InvalidTxIndex) Init() error {
	// Data compatibility migration
	err:=dropOldInvalidTx(idx.consensus.DatabaseContext())
	if err != nil {
		return err
	}
	//
	store := idx.consensus.InvalidTxIndexStore()
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
					return idx.caughtUpFrom(uint(tipOrder+1))
				}
			}
			return fmt.Errorf("vm block index(%s:%d) is out of synchronization(%s:%d) and can only be deleted and rebuilt:index --dropvmblock",
				tipHash, tipOrder, mainHash, mainOrder)
		}
		log.Info(fmt.Sprintf("Current %s tip:%s,%d",idx.Name(),tipHash.String(),tipOrder))
	}
	return nil
}

func (idx *InvalidTxIndex) caughtUpFrom(startOrder uint) error {
	store := idx.consensus.InvalidTxIndexStore()
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
		log.Info(fmt.Sprintf("Start caught up %s from (order:%d) to tip(hash:%s,order:%d)",idx.Name(),startOrder, mainHash, mainOrder))
		logLvl := l.Glogger().GetVerbosity()
		bar := progressbar.Default(int64(mainOrder-startOrder), fmt.Sprintf("%s:",idx.Name()))
		l.Glogger().Verbosity(l.LvlCrit)
		for i := uint(startOrder); i <= mainOrder; i++ {
			bar.Add(1)
			if i == 0 {
				continue
			}
			var block *types.SerializedBlock
			var blk model.Block
			err := idx.consensus.DatabaseContext().View(func(dbTx database.Tx) error {
				var er error
				block, blk, er= bc.DBFetchBlockByOrder(dbTx, uint64(i))
				return er
			})
			if err != nil {
				return err
			}
			if !blk.GetStatus().KnownInvalid() {
				continue
			}
			err=idx.ConnectBlock(uint64(blk.GetID()),block)
			if err != nil {
				return err
			}
		}
		l.Glogger().Verbosity(logLvl)
	}
	log.Info(fmt.Sprintf("Current %s tip:%s,%d",idx.Name(),mainHash.String(),mainOrder))
	return idx.UpdateMainTip(mainHash,uint64(mainOrder))
}

func (idx *InvalidTxIndex) ConnectBlock(bid uint64,block *types.SerializedBlock) error {
	store := idx.consensus.InvalidTxIndexStore()
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.Stage(stagingArea,bid,block)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func (idx *InvalidTxIndex) DisconnectBlock(bid uint64,block *types.SerializedBlock) error {
	store := idx.consensus.InvalidTxIndexStore()
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.Delete(stagingArea, bid,block)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func (idx *InvalidTxIndex) UpdateMainTip(bh *hash.Hash,order uint64) error {
	store := idx.consensus.InvalidTxIndexStore()
	if store == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	store.StageTip(stagingArea, bh,order)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func (idx *InvalidTxIndex) Get(txid *hash.Hash) (*types.Transaction, error) {
	store := idx.consensus.InvalidTxIndexStore()
	if store == nil {
		return nil,fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	return store.Get(stagingArea,txid)
}

func (idx *InvalidTxIndex) GetIdByHash(h *hash.Hash) (*hash.Hash, error) {
	store := idx.consensus.InvalidTxIndexStore()
	if store == nil {
		return nil,fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	return store.GetIdByHash(stagingArea,h)
}

func NewInvalidTxIndex(consensus model.Consensus) *InvalidTxIndex {
	log.Info(fmt.Sprintf("%s is enabled", invalidTxIndexName))
	return &InvalidTxIndex{
		consensus: consensus,
	}
}

// TODO: Discard in the future
func dropOldInvalidTx(db database.DB) error {
	var (
		itxIndexKey             = []byte("invalid_txbyhashidx")
		itxidByTxhashBucketName = []byte("invalid_txidbytxhash")
	)
	return db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()
		if meta.Bucket(itxIndexKey) != nil {
			err := meta.DeleteBucket(itxIndexKey)
			if err != nil {
				return err
			}
		}
		if meta.Bucket(itxidByTxhashBucketName) != nil {
			return meta.DeleteBucket(itxidByTxhashBucketName)
		}
		return nil
	})
}