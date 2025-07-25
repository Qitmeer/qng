package main

import (
	"github.com/Qitmeer/qng/v2/core/blockchain"
	"github.com/Qitmeer/qng/v2/database/legacydb"
)

type INode interface {
	BlockChain() *blockchain.BlockChain
	DB() legacydb.DB
}
