package chaindb

import (
	"bytes"
	"crypto/rand"
	"github.com/Qitmeer/qng/common/util"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/services/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/dbtest"
	elog "github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slices"
	"os"
	"testing"
	"time"
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

func BenchmarkIOLevelDB(b *testing.B) {
	doBenchmarkIO(b, "leveldb")
}

func BenchmarkIOPebbleDB(b *testing.B) {
	doBenchmarkIO(b, "pebble")
}

func doBenchmarkIO(b *testing.B, dbtype string) {
	dataDir, err := os.MkdirTemp("", "data_"+dbtype+"_*")
	if err != nil {
		b.Fatal(err)
	}
	log.Info("benchmark", "dbtype", dbtype, "datadir", dataDir)

	cfg := common.DefaultConfig("")
	l.Glogger().Verbosity(l.LvlCrit)
	eth.InitLog(elog.LvlCrit.String(), cfg.DebugPrintOrigins)
	BenchDatabaseSuite(b, func() ethdb.KeyValueStore {
		cfg.DataDir = dataDir
		cfg.DbType = dbtype
		cdb, err := NewNaked(cfg)
		if err != nil {
			b.Fatal(err)
		}
		return cdb.DB()
	})
	if util.FileExists(dataDir) {
		err = os.RemoveAll(dataDir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchDatabaseSuite runs a suite of benchmarks against a KeyValueStore database
// implementation.
func BenchDatabaseSuite(b *testing.B, New func() ethdb.KeyValueStore) {
	var (
		keys, vals   = makeDataset(1000, 32, 32, false)
		sKeys, sVals = makeDataset(1000, 32, 32, true)
	)
	// Run benchmarks sequentially
	b.Run("Write", func(b *testing.B) {
		benchWrite := func(b *testing.B, keys, vals [][]byte) {
			b.ResetTimer()

			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
		}
		b.Run("WriteSorted", func(b *testing.B) {
			benchWrite(b, sKeys, sVals)
		})
		b.Run("WriteRandom", func(b *testing.B) {
			benchWrite(b, keys, vals)
		})
	})
	b.Run("Read", func(b *testing.B) {
		benchRead := func(b *testing.B, keys, vals [][]byte) {
			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
			b.ResetTimer()

			for i := 0; i < len(keys); i++ {
				db.Get(keys[i])
			}
		}
		b.Run("ReadSorted", func(b *testing.B) {
			benchRead(b, sKeys, sVals)
		})
		b.Run("ReadRandom", func(b *testing.B) {
			benchRead(b, keys, vals)
		})
	})
	b.Run("Iteration", func(b *testing.B) {
		benchIteration := func(b *testing.B, keys, vals [][]byte) {
			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
			b.ResetTimer()

			it := db.NewIterator(nil, nil)
			for it.Next() {
			}
			it.Release()
		}
		b.Run("IterationSorted", func(b *testing.B) {
			benchIteration(b, sKeys, sVals)
		})
		b.Run("IterationRandom", func(b *testing.B) {
			benchIteration(b, keys, vals)
		})
	})
	b.Run("BatchWrite", func(b *testing.B) {
		benchBatchWrite := func(b *testing.B, keys, vals [][]byte) {
			b.ResetTimer()

			db := New()
			defer db.Close()

			batch := db.NewBatch()
			for i := 0; i < len(keys); i++ {
				batch.Put(keys[i], vals[i])
			}
			batch.Write()
		}
		b.Run("BenchWriteSorted", func(b *testing.B) {
			benchBatchWrite(b, sKeys, sVals)
		})
		b.Run("BenchWriteRandom", func(b *testing.B) {
			benchBatchWrite(b, keys, vals)
		})
	})
}

// randomHash generates a random blob of data and returns it as a hash.
func randBytes(len int) []byte {
	buf := make([]byte, len)
	if n, err := rand.Read(buf); n != len || err != nil {
		panic(err)
	}
	return buf
}

func makeDataset(size, ksize, vsize int, order bool) ([][]byte, [][]byte) {
	var keys [][]byte
	var vals [][]byte
	for i := 0; i < size; i += 1 {
		keys = append(keys, randBytes(ksize))
		vals = append(vals, randBytes(vsize))
	}
	if order {
		slices.SortFunc(keys, func(a, b []byte) int { return bytes.Compare(a, b) })
	}
	return keys, vals
}
