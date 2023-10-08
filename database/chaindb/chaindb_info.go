package chaindb

import (
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/rawdb"
)

func (cdb *ChainDB) GetInfo() (*common.DatabaseInfo, error) {
	version := rawdb.ReadDatabaseVersion(cdb.db)
	if version == nil {
		return nil, nil
	}
	compver := rawdb.ReadDatabaseCompressionVersion(cdb.db)
	if compver == nil {
		return nil, nil
	}
	biver := rawdb.ReadDatabaseBlockIndexVersion(cdb.db)
	if biver == nil {
		return nil, nil
	}
	create := rawdb.ReadDatabaseCreate(cdb.db)
	if create == nil {
		return nil, nil
	}
	return common.NewDatabaseInfo(*version, *compver, *biver, *create), nil
}

func (cdb *ChainDB) PutInfo(di *common.DatabaseInfo) error {
	batch := cdb.db.NewBatch()
	err := rawdb.WriteDatabaseVersion(batch, di.Version())
	if err != nil {
		return err
	}
	err = rawdb.WriteDatabaseCompressionVersion(batch, di.CompVer())
	if err != nil {
		return err
	}
	err = rawdb.WriteDatabaseBlockIndexVersion(batch, di.BidxVer())
	if err != nil {
		return err
	}
	err = rawdb.WriteDatabaseCreate(batch, di.Created())
	if err != nil {
		return err
	}
	return batch.Write()
}
