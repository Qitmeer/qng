package index

import (
	"fmt"
	"github.com/Qitmeer/qng/common/staging"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database"
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
		stagingArea := model.NewStagingArea()
		vmbiStore.StageTip(stagingArea, mainHash, uint64(mainOrder))
		err := staging.CommitAllChanges(idx.consensus.DatabaseContext(), stagingArea)
		if err != nil {
			return err
		}
		log.Trace(fmt.Sprintf("Start rebuilding vm block index to tip(hash:%s,order:%d)", mainHash, mainOrder))
		for i := uint(0); i <= mainOrder; i++ {
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
	} else {
		tipOrder, tipHash, err := vmbiStore.Tip(model.NewStagingArea())
		if err != nil {
			return err
		}
		if tipOrder != uint64(mainOrder) || mainHash.IsEqual(tipHash) {
			return fmt.Errorf("vm block index(%s:%d) is out of synchronization(%s:%d) and can only be deleted and rebuilt:index --dropvmblock",
				tipHash, tipOrder, mainHash, mainOrder)
		}
	}
	return nil
}

func (idx *VMBlockIndex) Name() string {
	return vmblockIndexName
}

func (idx *VMBlockIndex) ConnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos [][]byte, blk model.Block) error {
	return nil
}

func (idx *VMBlockIndex) DisconnectBlock(dbTx database.Tx, block *types.SerializedBlock, stxos [][]byte) error {

	return nil
}

func NewVMBlockIndex(consensus model.Consensus) *VMBlockIndex {
	log.Info(fmt.Sprintf("%s is enabled", vmblockIndexName))
	return &VMBlockIndex{
		consensus: consensus,
	}
}

func DropVMBlockIndex(db database.DB, interrupt <-chan struct{}) error {
	log.Info("Start drop vm block index")
	return nil
}
