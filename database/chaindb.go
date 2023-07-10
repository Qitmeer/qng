package database

import "github.com/ethereum/go-ethereum/ethdb"

type ChainDB struct {
	db ethdb.Database
}

func (cdb *ChainDB) Name() string {
	return "Chain DB"
}
