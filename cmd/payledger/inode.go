package main

import (
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/database/legacydb"
)

type INode interface {
	BlockChain() *blockchain.BlockChain
	DB() legacydb.DB
}
