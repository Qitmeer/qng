package main

import (
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/database"
)

type INode interface {
	BlockChain() *blockchain.BlockChain
	DB() database.DB
}
