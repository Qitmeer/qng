package chaindb

import (
	"github.com/Qitmeer/qng/v2/database/common"
	"github.com/Qitmeer/qng/v2/database/rawdb"
	"github.com/Qitmeer/qng/v2/params"
)

func (cdb *ChainDB) TryUpgrade(di *common.DatabaseInfo, interrupt <-chan struct{}) error {
	if !params.ActiveNetParams.ConsensusConfig.Type().IsPoW() {
		return nil
	}
	if di.Version() == common.CurrentDatabaseVersion {
		// To fix old data, re index old genesis transaction data.
		txId := params.ActiveNetParams.GenesisBlock.Transactions()[0].Hash()
		_, blockHash, err := cdb.GetTxIdxEntry(txId, false)
		if err != nil {
			return err
		}
		if blockHash == nil {
			log.Info("Re index genesis transaction for database")
			return rawdb.WriteTxLookupEntriesByBlock(cdb.db, params.ActiveNetParams.GenesisBlock, 0)
		}
		return nil
	}
	return nil
}
