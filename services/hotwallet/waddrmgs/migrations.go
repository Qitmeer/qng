package waddrmgr

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/log"
	chaincfg "github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb/migration"
	"github.com/Qitmeer/qng/services/hotwallet/wtxmgr"
	"time"
)

// versions is a list of the different database versions. The last entry should
// reflect the latest database state. If the database happens to be at a version
// number lower than the latest, migrations will be performed in order to catch
// it up.
var versions = []migration.Version{
	{
		Number:    2,
		Migration: upgradeToVersion2,
	},
	{
		Number:    6,
		Migration: populateBirthdayBlock,
	},
	{
		Number:    7,
		Migration: resetSyncedBlockToBirthday,
	},
}

// getLatestVersion returns the version number of the latest database version.
func getLatestVersion() uint32 {
	return versions[len(versions)-1].Number
}

// MigrationManager is an implementation of the migration.Manager interface that
// will be used to handle migrations for the address manager. It exposes the
// necessary parameters required to successfully perform migrations.
type MigrationManager struct {
	ns walletdb.ReadWriteBucket
}

// A compile-time assertion to ensure that MigrationManager implements the
// migration.Manager interface.
var _ migration.Manager = (*MigrationManager)(nil)

// Name returns the name of the service we'll be attempting to upgrade.
//
// NOTE: This method is part of the migration.Manager interface.
func (m *MigrationManager) Name() string {
	return "wallet address manager"
}

// Namespace returns the top-level bucket of the service.
//
// NOTE: This method is part of the migration.Manager interface.
func (m *MigrationManager) Namespace() walletdb.ReadWriteBucket {
	return m.ns
}

// CurrentVersion returns the current version of the service's database.
//
// NOTE: This method is part of the migration.Manager interface.
func (m *MigrationManager) CurrentVersion(ns walletdb.ReadBucket) (uint32, error) {
	if ns == nil {
		ns = m.ns
	}
	return fetchManagerVersion(ns)
}

// SetVersion sets the version of the service's database.
//
// NOTE: This method is part of the migration.Manager interface.
func (m *MigrationManager) SetVersion(ns walletdb.ReadWriteBucket,
	version uint32) error {

	if ns == nil {
		ns = m.ns
	}
	return putManagerVersion(m.ns, version)
}

// Versions returns all of the available database versions of the service.
//
// NOTE: This method is part of the migration.Manager interface.
func (m *MigrationManager) Versions() []migration.Version {
	return versions
}

// upgradeToVersion2 upgrades the database from version 1 to version 2
// 'usedAddrBucketName' a bucket for storing addrs flagged as marked is
// initialized and it will be updated on the next rescan.
func upgradeToVersion2(ns walletdb.ReadWriteBucket) error {
	currentMgrVersion := uint32(2)

	_, err := ns.CreateBucketIfNotExists(usedAddrBucketName)
	if err != nil {
		str := "failed to create used addresses bucket"
		return managerError(ErrDatabase, str, err)
	}

	return putManagerVersion(ns, currentMgrVersion)
}

// populateBirthdayBlock is a migration that attempts to populate the birthday
// block of the wallet. This is needed so that in the event that we need to
// perform a rescan of the wallet, we can do so starting from this block, rather
// than from the genesis block.
//
// NOTE: This migration cannot guarantee the correctness of the birthday block
// being set as we do not store block timestamps, so a sanity check must be done
// upon starting the wallet to ensure we do not potentially miss any relevant
// events when rescanning.
func populateBirthdayBlock(ns walletdb.ReadWriteBucket) error {
	// We'll need to jump through some hoops in order to determine the
	// corresponding block height for our birthday timestamp. Since we do
	// not store block timestamps, we'll need to estimate our height by
	// looking at the genesis timestamp and assuming a block occurs every 10
	// minutes. This can be unsafe, and cause us to actually miss on-chain
	// events, so a sanity check is done before the wallet attempts to sync
	// itself.
	//
	// We'll start by fetching our birthday timestamp.
	birthdayTimestamp, err := fetchBirthday(ns)
	if err != nil {
		return fmt.Errorf("unable to fetch birthday timestamp: %v", err)
	}

	log.Info("Setting the wallet's birthday block from timestamp=%v",
		birthdayTimestamp)

	// Now, we'll need to determine the timestamp of the genesis block for
	// the corresponding chain.
	genesisHash, err := fetchBlockHash(ns, 0)
	if err != nil {
		return fmt.Errorf("unable to fetch genesis block hash: %v", err)
	}

	var genesisTimestamp time.Time
	switch *genesisHash {
	case *chaincfg.MainNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.MainNetParams.GenesisBlock.Header.Timestamp

	case *chaincfg.TestNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.TestNetParams.GenesisBlock.Header.Timestamp

	case *chaincfg.PrivNetParams.GenesisHash:
		genesisTimestamp =
			chaincfg.PrivNetParams.GenesisBlock.Header.Timestamp

	default:
		return fmt.Errorf("unknown genesis hash %v", genesisHash)
	}

	// With the timestamps retrieved, we can estimate a block height by
	// taking the difference between them and dividing by the average block
	// time (10 minutes).
	birthdayHeight := int32(birthdayTimestamp.Sub(genesisTimestamp).Seconds() / 600)

	// Now that we have the height estimate, we can fetch the corresponding
	// block and set it as our birthday block.
	birthdayHash, err := fetchBlockHash(ns, uint32(birthdayHeight))

	// To ensure we record a height that is known to us from the chain,
	// we'll make sure this height estimate can be found. Otherwise, we'll
	// continue subtracting a day worth of blocks until we can find one.
	for IsError(err, ErrBlockNotFound) {
		birthdayHeight -= 144
		if birthdayHeight < 0 {
			birthdayHeight = 0
		}
		birthdayHash, err = fetchBlockHash(ns, uint32(birthdayHeight))
	}
	if err != nil {
		return err
	}

	log.Info("Estimated birthday block from timestamp=%v: height=%d, "+
		"hash=%v", birthdayTimestamp, birthdayHeight, birthdayHash)

	// NOTE: The timestamp of the birthday block isn't set since we do not
	// store each block's timestamp.
	return putBirthdayBlock(ns, BlockStamp{
		Order: uint32(birthdayHeight),
		Hash:  *birthdayHash,
	})
}

// resetSyncedBlockToBirthday is a migration that resets the wallet's currently
// synced block to its birthday block. This essentially serves as a migration to
// force a rescan of the wallet.
func resetSyncedBlockToBirthday(ns walletdb.ReadWriteBucket) error {
	syncBucket := ns.NestedReadWriteBucket(wtxmgr.BucketSync)
	if syncBucket == nil {
		return errors.New("sync bucket does not exist")
	}

	birthdayBlock, err := FetchBirthdayBlock(ns)
	if err != nil {
		return err
	}

	return PutSyncedTo(ns, &birthdayBlock)
}
