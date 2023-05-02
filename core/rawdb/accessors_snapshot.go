package rawdb

import (
	"encoding/binary"
	"github.com/Qitmeer/qng/common/hash"

	"github.com/ethereum/go-ethereum/ethdb"
)

// ReadSnapshotDisabled retrieves if the snapshot maintenance is disabled.
func ReadSnapshotDisabled(db ethdb.KeyValueReader) bool {
	disabled, _ := db.Has(snapshotDisabledKey)
	return disabled
}

// WriteSnapshotDisabled stores the snapshot pause flag.
func WriteSnapshotDisabled(db ethdb.KeyValueWriter) {
	if err := db.Put(snapshotDisabledKey, []byte("no")); err != nil {
		log.Crit("Failed to store snapshot disabled flag", "err", err)
	}
}

// DeleteSnapshotDisabled deletes the flag keeping the snapshot maintenance disabled.
func DeleteSnapshotDisabled(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotDisabledKey); err != nil {
		log.Crit("Failed to remove snapshot disabled flag", "err", err)
	}
}

// ReadSnapshotRoot retrieves the root of the block whose state is contained in
// the persisted snapshot.
func ReadSnapshotRoot(db ethdb.KeyValueReader) *hash.Hash {
	data, _ := db.Get(SnapshotRootKey)
	if len(data) != hash.HashSize {
		return nil
	}
	h, err := hash.NewHash(data)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return h
}

// WriteSnapshotRoot stores the root of the block whose state is contained in
// the persisted snapshot.
func WriteSnapshotRoot(db ethdb.KeyValueWriter, root *hash.Hash) {
	if err := db.Put(SnapshotRootKey, root.Bytes()); err != nil {
		log.Error("Failed to store snapshot root", "err", err)
	}
}

// DeleteSnapshotRoot deletes the hash of the block whose state is contained in
// the persisted snapshot. Since snapshots are not immutable, this  method can
// be used during updates, so a crash or failure will mark the entire snapshot
// invalid.
func DeleteSnapshotRoot(db ethdb.KeyValueWriter) {
	if err := db.Delete(SnapshotRootKey); err != nil {
		log.Error("Failed to remove snapshot root", "err", err)
	}
}

func ReadBlockOrderSnapshot(db ethdb.KeyValueReader, order uint64) *uint64 {
	data, err := db.Get(blockOrderKey(order))
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	id := binary.BigEndian.Uint64(data)
	return &id
}

func WriteBlockOrderSnapshot(db ethdb.KeyValueWriter, order uint64, id uint64) error {
	return db.Put(blockOrderKey(order), encodeBlockID(id))
}

func DeleteBlockOrderSnapshot(db ethdb.KeyValueWriter, order uint64) error {
	return db.Delete(blockOrderKey(order))
}

func ReadSnapshotJournal(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotJournalKey)
	return data
}

func WriteSnapshotJournal(db ethdb.KeyValueWriter, journal []byte) {
	if err := db.Put(snapshotJournalKey, journal); err != nil {
		log.Error("Failed to store snapshot journal", "err", err)
	}
}

// DeleteSnapshotJournal deletes the serialized in-memory diff layers saved at
// the last shutdown
func DeleteSnapshotJournal(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotJournalKey); err != nil {
		log.Error("Failed to remove snapshot journal", "err", err)
	}
}

// ReadSnapshotGenerator retrieves the serialized snapshot generator saved at
// the last shutdown.
func ReadSnapshotGenerator(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotGeneratorKey)
	return data
}

// WriteSnapshotGenerator stores the serialized snapshot generator to save at
// shutdown.
func WriteSnapshotGenerator(db ethdb.KeyValueWriter, generator []byte) {
	if err := db.Put(snapshotGeneratorKey, generator); err != nil {
		log.Error("Failed to store snapshot generator", "err", err)
	}
}

// DeleteSnapshotGenerator deletes the serialized snapshot generator saved at
// the last shutdown
func DeleteSnapshotGenerator(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotGeneratorKey); err != nil {
		log.Error("Failed to remove snapshot generator", "err", err)
	}
}

// ReadSnapshotRecoveryNumber retrieves the block number of the last persisted
// snapshot layer.
func ReadSnapshotRecoveryNumber(db ethdb.KeyValueReader) *uint64 {
	data, _ := db.Get(snapshotRecoveryKey)
	if len(data) == 0 {
		return nil
	}
	if len(data) != 8 {
		return nil
	}
	number := binary.BigEndian.Uint64(data)
	return &number
}

// WriteSnapshotRecoveryNumber stores the block number of the last persisted
// snapshot layer.
func WriteSnapshotRecoveryNumber(db ethdb.KeyValueWriter, number uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], number)
	if err := db.Put(snapshotRecoveryKey, buf[:]); err != nil {
		log.Error("Failed to store snapshot recovery number", "err", err)
	}
}

// DeleteSnapshotRecoveryNumber deletes the block number of the last persisted
// snapshot layer.
func DeleteSnapshotRecoveryNumber(db ethdb.KeyValueWriter) {
	if err := db.Delete(snapshotRecoveryKey); err != nil {
		log.Error("Failed to remove snapshot recovery number", "err", err)
	}
}

// ReadSnapshotSyncStatus retrieves the serialized sync status saved at shutdown.
func ReadSnapshotSyncStatus(db ethdb.KeyValueReader) []byte {
	data, _ := db.Get(snapshotSyncStatusKey)
	return data
}

// WriteSnapshotSyncStatus stores the serialized sync status to save at shutdown.
func WriteSnapshotSyncStatus(db ethdb.KeyValueWriter, status []byte) {
	if err := db.Put(snapshotSyncStatusKey, status); err != nil {
		log.Error("Failed to store snapshot sync status", "err", err)
	}
}
