package node

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/params"
	"testing"
)

func TestNodeCloseClosesDB(t *testing.T) {
	node, err := NewNode(&config.Config{}, nil, params.ActiveNetParams.Params, nil)
	if err != nil {
		t.Fatal("node:", err)
	}
	defer node.Stop()

	db, err := node.OpenDatabase("mydb", 0, 0, "", false)
	if err != nil {
		t.Fatal("can't open DB:", err)
	}
	if err = db.Put([]byte{}, []byte{}); err != nil {
		t.Fatal("can't Put on open DB:", err)
	}

	node.CloseDatabases()
	if err = db.Put([]byte{}, []byte{}); err == nil {
		t.Fatal("Put succeeded after node is closed")
	}
}
