package chaindb

import (
	"github.com/Qitmeer/qng/services/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/dbtest"
	"testing"
)

func TestChainCloseClosesDB(t *testing.T) {
	cfg := common.DefaultConfig("")
	cfg.DataDir = ""
	cdb, err := NewNaked(cfg)
	if err != nil {
		t.Fatal("node:", err)
	}
	defer cdb.Close()

	if err = cdb.db.Put([]byte{}, []byte{}); err != nil {
		t.Fatal("can't Put on open DB:", err)
	}

	cdb.CloseDatabases()
	if err = cdb.db.Put([]byte{}, []byte{}); err == nil {
		t.Fatal("Put succeeded after node is closed")
	}
}

func BenchmarkLevelDB(b *testing.B) {
	dbtest.BenchDatabaseSuite(b, func() ethdb.KeyValueStore {
		cfg := common.DefaultConfig("")
		cfg.DataDir = ""
		cfg.DbType = "leveldb"
		cdb, err := NewNaked(cfg)
		if err != nil {
			b.Fatal(err)
		}
		return cdb.DB()
	})
}

func BenchmarkPebbleDB(b *testing.B) {
	dbtest.BenchDatabaseSuite(b, func() ethdb.KeyValueStore {
		cfg := common.DefaultConfig("")
		cfg.DataDir = ""
		cfg.DbType = "pebble"
		cdb, err := NewNaked(cfg)
		if err != nil {
			b.Fatal(err)
		}
		return cdb.DB()
	})
}

func TestChainDBBatch(t *testing.T) {
	cfg := common.DefaultConfig("")
	cfg.DataDir = ""
	cdb, err := NewNaked(cfg)
	if err != nil {
		t.Fatal("node:", err)
	}
	defer cdb.Close()

	batch := cdb.db.NewBatch()

	k := []byte("k1")
	v := []byte("v1")

	if err = batch.Put(k, v); err != nil {
		t.Fatal("batch:", err)
	}

	var exist bool
	exist, err = cdb.db.Has(k)
	if err != nil {
		t.Fatal(err)
	}
	if exist {
		t.Fatalf("want absent,but exist")
	}

	err = batch.Write()
	if err != nil {
		t.Fatal(err)
	}

	exist, err = cdb.db.Has(k)
	if err != nil {
		t.Fatal(err)
	}
	if !exist {
		t.Fatalf("want exist,but absent")
	}
}
