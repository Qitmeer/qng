package rawdb

import (
	"encoding/binary"
	"testing"
)

func TestKeyLengthIterator(t *testing.T) {
	db := NewMemoryDatabase()

	keyLen := 8
	expectedKeys := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		key := make([]byte, keyLen)
		binary.BigEndian.PutUint64(key, uint64(i))
		if err := db.Put(key, []byte{0x1}); err != nil {
			t.Fatal(err)
		}
		expectedKeys[string(key)] = struct{}{}

		longerKey := make([]byte, keyLen*2)
		binary.BigEndian.PutUint64(longerKey, uint64(i))
		if err := db.Put(longerKey, []byte{0x1}); err != nil {
			t.Fatal(err)
		}
	}

	it := NewKeyLengthIterator(db.NewIterator(nil, nil), keyLen)
	for it.Next() {
		key := it.Key()
		_, exists := expectedKeys[string(key)]
		if !exists {
			t.Fatalf("Found unexpected key %d", binary.BigEndian.Uint64(key))
		}
		delete(expectedKeys, string(key))
		if len(key) != keyLen {
			t.Fatalf("Found unexpected key in key length iterator with length %d", len(key))
		}
	}

	if len(expectedKeys) != 0 {
		t.Fatalf("Expected all keys of length %d to be removed from expected keys during iteration", keyLen)
	}
}
