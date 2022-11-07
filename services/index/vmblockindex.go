package index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/consensus/store/vm_block_index"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"github.com/schollz/progressbar/v3"
)

const (
	vmblockIndexName = "vm block index"
)

type VMBlockIndex struct {
	consensus model.Consensus
}

func (idx *VMBlockIndex) Init() error {
	vmbiStore := idx.consensus.VMBlockIndexStore()
	if vmbiStore == nil {
		return fmt.Errorf("No vm block index store")
	}
	bc := idx.consensus.BlockChain()
	mainOrder := bc.GetMainOrder()
	mainHash := bc.GetBlockHashByOrder(mainOrder)
	if mainHash == nil {
		return fmt.Errorf("No block in order:%d", mainOrder)
	}
	if vmbiStore.IsEmpty() {
		log.Trace(fmt.Sprintf("Start rebuilding vm block index to tip(hash:%s,order:%d)", mainHash, mainOrder))
		if mainOrder > 0 {
			logLvl := l.Glogger().GetVerbosity()
			bar := progressbar.Default(int64(mainOrder), "vmblock index:")
			l.Glogger().Verbosity(l.LvlCrit)
			for i := uint(1); i <= mainOrder; i++ {
				bar.Add(1)
				bh := bc.GetBlockHashByOrder(i)
				if bh == nil {
					return fmt.Errorf("No block in order:%d", i)
				}
				bid := bc.(*blockchain.BlockChain).VMService.GetBlockID(bh)
				if bid == 0 {
					continue
				}
				stagingArea := model.NewStagingArea()
				vmbiStore.Stage(stagingArea, bid, bh)
				err := staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
				if err != nil {
					return err
				}
			}
			l.Glogger().Verbosity(logLvl)
		}
		stagingArea := model.NewStagingArea()
		vmbiStore.StageTip(stagingArea, mainHash, uint64(mainOrder))
		err := staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
		if err != nil {
			return err
		}
	} else {
		tipOrder, tipHash, err := vmbiStore.Tip(model.NewStagingArea())
		if err != nil {
			return err
		}
		if tipOrder != uint64(mainOrder) || !mainHash.IsEqual(tipHash) {
			return fmt.Errorf("vm block index(%s:%d) is out of synchronization(%s:%d) and can only be deleted and rebuilt:index --dropvmblock",
				tipHash, tipOrder, mainHash, mainOrder)
		}
		log.Info(fmt.Sprintf("vmblock index cur tip:%s,%d",tipHash.String(),tipOrder))
	}
	return nil
}

func (idx *VMBlockIndex) Name() string {
	return vmblockIndexName
}

func (idx *VMBlockIndex) ConnectBlock(bh *hash.Hash,vmbid uint64) error {
	vmbiStore := idx.consensus.VMBlockIndexStore()
	if vmbiStore == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	vmbiStore.Stage(stagingArea, vmbid, bh)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func (idx *VMBlockIndex) DisconnectBlock(bh *hash.Hash,vmbid uint64) error {
	vmbiStore := idx.consensus.VMBlockIndexStore()
	if vmbiStore == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	vmbiStore.Delete(stagingArea, vmbid)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func (idx *VMBlockIndex) UpdateMainTip(bh *hash.Hash,order uint64) error {
	vmbiStore := idx.consensus.VMBlockIndexStore()
	if vmbiStore == nil {
		return fmt.Errorf("No vm block index store")
	}
	stagingArea := model.NewStagingArea()
	vmbiStore.StageTip(stagingArea, bh,order)
	return staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
}

func NewVMBlockIndex(consensus model.Consensus) *VMBlockIndex {
	log.Info(fmt.Sprintf("%s is enabled", vmblockIndexName))
	return &VMBlockIndex{
		consensus: consensus,
	}
}

func DropVMBlockIndex(db database.DB, interrupt <-chan struct{}) error {
	log.Info("Start drop vm block index")
	vmbiStore, err := vm_block_index.New(db, 10, false)
	if err != nil {
		return err
	}
	if vmbiStore.IsEmpty() {
		return fmt.Errorf("No data needs to be deleted")
	}
	tipOrder, tipHash, err := vmbiStore.Tip(model.NewStagingArea())
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("All vmblock index at (%s,%d) will be deleted",tipHash,tipOrder))
	return vmbiStore.Clean()
}
