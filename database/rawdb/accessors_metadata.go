package rawdb

import (
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

func ReadDatabaseVersion(db ethdb.KeyValueReader) *uint32 {
	var version uint32

	enc, _ := db.Get(VersionKey)
	if len(enc) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(enc, &version); err != nil {
		return nil
	}

	return &version
}

func WriteDatabaseVersion(db ethdb.KeyValueWriter, version uint32) error {
	enc, err := rlp.EncodeToBytes(version)
	if err != nil {
		log.Error("Failed to encode database version", "err", err)
		return err
	}
	err = db.Put(VersionKey, enc)
	if err != nil {
		log.Error("Failed to store the database version", "err", err)
		return err
	}
	return nil
}

func ReadDatabaseCompressionVersion(db ethdb.KeyValueReader) *uint32 {
	var version uint32

	enc, _ := db.Get(CompressionVersionKey)
	if len(enc) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(enc, &version); err != nil {
		return nil
	}

	return &version
}

func WriteDatabaseCompressionVersion(db ethdb.KeyValueWriter, version uint32) error {
	enc, err := rlp.EncodeToBytes(version)
	if err != nil {
		log.Error("Failed to encode database compression version", "err", err)
		return err
	}
	err = db.Put(CompressionVersionKey, enc)
	if err != nil {
		log.Error("Failed to store the database compression version", "err", err)
		return err
	}
	return nil
}

func ReadDatabaseBlockIndexVersion(db ethdb.KeyValueReader) *uint32 {
	var version uint32

	enc, _ := db.Get(BlockIndexVersionKey)
	if len(enc) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(enc, &version); err != nil {
		return nil
	}

	return &version
}

func WriteDatabaseBlockIndexVersion(db ethdb.KeyValueWriter, version uint32) error {
	enc, err := rlp.EncodeToBytes(version)
	if err != nil {
		log.Error("Failed to encode database block index version", "err", err)
		return err
	}
	err = db.Put(BlockIndexVersionKey, enc)
	if err != nil {
		log.Error("Failed to store the database block index version", "err", err)
		return err
	}
	return nil
}

func ReadDatabaseCreate(db ethdb.KeyValueReader) *time.Time {
	var create uint64

	enc, _ := db.Get(CreatedKey)
	if len(enc) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(enc, &create); err != nil {
		return nil
	}
	ct := time.Unix(int64(create), 0)
	return &ct
}

func WriteDatabaseCreate(db ethdb.KeyValueWriter, create time.Time) error {
	enc, err := rlp.EncodeToBytes(uint64(create.Unix()))
	if err != nil {
		log.Error("Failed to encode database create time", "err", err)
		return err
	}
	err = db.Put(CreatedKey, enc)
	if err != nil {
		log.Error("Failed to store the database create time", "err", err)
		return err
	}
	return nil
}

// crashList is a list of unclean-shutdown-markers, for rlp-encoding to the
// database
type crashList struct {
	Discarded uint64   // how many ucs have we deleted
	Recent    []uint64 // unix timestamps of 10 latest unclean shutdowns
}

const crashesToKeep = 10

// PushUncleanShutdownMarker appends a new unclean shutdown marker and returns
// the previous data
// - a list of timestamps
// - a count of how many old unclean-shutdowns have been discarded
func PushUncleanShutdownMarker(db ethdb.KeyValueStore) ([]uint64, uint64, error) {
	var uncleanShutdowns crashList
	// Read old data
	if data, err := db.Get(uncleanShutdownKey); err != nil {
		log.Warn("Error reading unclean shutdown markers", "error", err)
	} else if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
		return nil, 0, err
	}
	var discarded = uncleanShutdowns.Discarded
	var previous = make([]uint64, len(uncleanShutdowns.Recent))
	copy(previous, uncleanShutdowns.Recent)
	// Add a new (but cap it)
	uncleanShutdowns.Recent = append(uncleanShutdowns.Recent, uint64(time.Now().Unix()))
	if count := len(uncleanShutdowns.Recent); count > crashesToKeep+1 {
		numDel := count - (crashesToKeep + 1)
		uncleanShutdowns.Recent = uncleanShutdowns.Recent[numDel:]
		uncleanShutdowns.Discarded += uint64(numDel)
	}
	// And save it again
	data, _ := rlp.EncodeToBytes(uncleanShutdowns)
	if err := db.Put(uncleanShutdownKey, data); err != nil {
		log.Warn("Failed to write unclean-shutdown marker", "err", err)
		return nil, 0, err
	}
	return previous, discarded, nil
}

// PopUncleanShutdownMarker removes the last unclean shutdown marker
func PopUncleanShutdownMarker(db ethdb.KeyValueStore) {
	var uncleanShutdowns crashList
	// Read old data
	if data, err := db.Get(uncleanShutdownKey); err != nil {
		log.Warn("Error reading unclean shutdown markers", "error", err)
	} else if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
		log.Error("Error decoding unclean shutdown markers", "error", err) // Should mos def _not_ happen
	}
	if l := len(uncleanShutdowns.Recent); l > 0 {
		uncleanShutdowns.Recent = uncleanShutdowns.Recent[:l-1]
	}
	data, _ := rlp.EncodeToBytes(uncleanShutdowns)
	if err := db.Put(uncleanShutdownKey, data); err != nil {
		log.Warn("Failed to clear unclean-shutdown marker", "err", err)
	}
}

// UpdateUncleanShutdownMarker updates the last marker's timestamp to now.
func UpdateUncleanShutdownMarker(db ethdb.KeyValueStore) {
	var uncleanShutdowns crashList
	// Read old data
	if data, err := db.Get(uncleanShutdownKey); err != nil {
		log.Warn("Error reading unclean shutdown markers", "error", err)
	} else if err := rlp.DecodeBytes(data, &uncleanShutdowns); err != nil {
		log.Warn("Error decoding unclean shutdown markers", "error", err)
	}
	// This shouldn't happen because we push a marker on Backend instantiation
	count := len(uncleanShutdowns.Recent)
	if count == 0 {
		log.Warn("No unclean shutdown marker to update")
		return
	}
	uncleanShutdowns.Recent[count-1] = uint64(time.Now().Unix())
	data, _ := rlp.EncodeToBytes(uncleanShutdowns)
	if err := db.Put(uncleanShutdownKey, data); err != nil {
		log.Warn("Failed to write unclean-shutdown marker", "err", err)
	}
}

// best chain state
func ReadBestChainState(db ethdb.Reader) []byte {
	data, err := db.Get(bestChainStateKey)
	if err != nil {
		if isErrWithoutNotFound(err) {
			log.Error("best chain state", "err", err.Error())
		}
		return nil
	}
	return data
}

func WriteBestChainState(db ethdb.KeyValueWriter, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(bestChainStateKey, data)
}

// estimatefee
func ReadEstimateFee(db ethdb.Reader) []byte {
	data, err := db.Get(EstimateFeeDatabaseKey)
	if err != nil {
		if isErrWithoutNotFound(err) {
			log.Error("estimate fee", "err", err.Error())
		}
		return nil
	}
	return data
}

func WriteEstimateFee(db ethdb.KeyValueWriter, data []byte) error {
	if len(data) <= 0 {
		return nil
	}
	return db.Put(EstimateFeeDatabaseKey, data)
}

func DeleteEstimateFee(db ethdb.KeyValueWriter) error {
	return db.Delete(EstimateFeeDatabaseKey)
}
