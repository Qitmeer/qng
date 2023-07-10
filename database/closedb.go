package database

import (
	"github.com/ethereum/go-ethereum/ethdb"
)

// closeTrackingDB wraps the Close method of a database. When the database is closed by the
// service, the wrapper removes it from the node's database map. This ensures that Node
// won't auto-close the database if it is closed by the service that opened it.
type closeTrackingDB struct {
	ethdb.Database
	c *ChainDB
}

func (db *closeTrackingDB) Close() error {
	db.c.lock.Lock()
	delete(db.c.databases, db)
	db.c.lock.Unlock()
	return db.Database.Close()
}
