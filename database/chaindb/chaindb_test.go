package chaindb

import (
	"github.com/Qitmeer/qng/config"
	"testing"
)

func TestChainCloseClosesDB(t *testing.T) {
	cdb, err := New(&config.Config{DevNextGDB: true})
	if err != nil {
		t.Fatal("node:", err)
	}
	defer cdb.Close()

	db, err := cdb.OpenDatabase("mydb", 0, 0, "", false)
	if err != nil {
		t.Fatal("can't open DB:", err)
	}
	if err = db.Put([]byte{}, []byte{}); err != nil {
		t.Fatal("can't Put on open DB:", err)
	}

	cdb.CloseDatabases()
	if err = db.Put([]byte{}, []byte{}); err == nil {
		t.Fatal("Put succeeded after node is closed")
	}
}
