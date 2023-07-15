package legacychaindb

import (
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/database/rawdb"
	"time"
)

func (cdb *LegacyChainDB) GetInfo() (*common.DatabaseInfo, error) {
	var di *common.DatabaseInfo
	err := cdb.db.View(func(dbTx legacydb.Tx) error {
		// Fetch the database versioning information.
		dbInfo, err := dbFetchDatabaseInfo(dbTx)
		if err != nil {
			return err
		}
		di = dbInfo
		return nil
	})
	return di, err
}

func (cdb *LegacyChainDB) PutInfo(di *common.DatabaseInfo) error {
	err := cdb.db.Update(func(dbTx legacydb.Tx) error {
		meta := dbTx.Metadata()

		// Create the bucket that houses information about the database's
		// creation and version.
		_, err := meta.CreateBucketIfNotExists(dbnamespace.BCDBInfoBucketName)
		if err != nil {
			return err
		}
		err = dbPutDatabaseInfo(dbTx, di)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}

// dbFetchDatabaseInfo uses an existing database transaction to fetch the
// database versioning and creation information.
func dbFetchDatabaseInfo(dbTx legacydb.Tx) (*common.DatabaseInfo, error) {
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.BCDBInfoBucketName)

	// Uninitialized state.
	if bucket == nil {
		return nil, nil
	}

	// Load the database version.
	var version uint32
	versionBytes := bucket.Get(rawdb.VersionKey)
	if versionBytes != nil {
		version = dbnamespace.ByteOrder.Uint32(versionBytes)
	}

	// Load the database compression version.
	var compVer uint32
	compVerBytes := bucket.Get(rawdb.CompressionVersionKey)
	if compVerBytes != nil {
		compVer = dbnamespace.ByteOrder.Uint32(compVerBytes)
	}

	// Load the database block index version.
	var bidxVer uint32
	bidxVerBytes := bucket.Get(rawdb.BlockIndexVersionKey)
	if bidxVerBytes != nil {
		bidxVer = dbnamespace.ByteOrder.Uint32(bidxVerBytes)
	}

	// Load the database creation date.
	var created time.Time
	createdBytes := bucket.Get(rawdb.CreatedKey)
	if createdBytes != nil {
		ts := dbnamespace.ByteOrder.Uint64(createdBytes)
		created = time.Unix(int64(ts), 0)
	}

	return common.NewDatabaseInfo(version, compVer, bidxVer, created), nil
}

// dbPutDatabaseInfo uses an existing database transaction to store the database
// information.
func dbPutDatabaseInfo(dbTx legacydb.Tx, dbi *common.DatabaseInfo) error {
	// uint32Bytes is a helper function to convert a uint32 to a byte slice
	// using the byte order specified by the database namespace.
	uint32Bytes := func(ui32 uint32) []byte {
		var b [4]byte
		dbnamespace.ByteOrder.PutUint32(b[:], ui32)
		return b[:]
	}

	// uint64Bytes is a helper function to convert a uint64 to a byte slice
	// using the byte order specified by the database namespace.
	uint64Bytes := func(ui64 uint64) []byte {
		var b [8]byte
		dbnamespace.ByteOrder.PutUint64(b[:], ui64)
		return b[:]
	}

	// Store the database version.
	meta := dbTx.Metadata()
	bucket := meta.Bucket(dbnamespace.BCDBInfoBucketName)
	err := bucket.Put(rawdb.VersionKey,
		uint32Bytes(dbi.Version()))
	if err != nil {
		return err
	}

	// Store the compression version.
	err = bucket.Put(rawdb.CompressionVersionKey,
		uint32Bytes(dbi.CompVer()))
	if err != nil {
		return err
	}

	// Store the block index version.
	err = bucket.Put(rawdb.BlockIndexVersionKey,
		uint32Bytes(dbi.BidxVer()))
	if err != nil {
		return err
	}

	// Store the database creation date.
	return bucket.Put(rawdb.CreatedKey,
		uint64Bytes(uint64(dbi.Created().Unix())))
}
