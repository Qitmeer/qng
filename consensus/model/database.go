package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
)

type DataBase interface {
	Name() string
	Init() error
	Close()
	Rebuild(mgr IndexManager) error
	GetInfo() (*common.DatabaseInfo, error)
	PutInfo(di *common.DatabaseInfo) error
	GetSpendJournal(bh *hash.Hash) ([]byte, error)
	PutSpendJournal(bh *hash.Hash, data []byte) error
	DeleteSpendJournal(bh *hash.Hash) error
	GetUtxo(key []byte) ([]byte, error)
	PutUtxo(key []byte, data []byte) error
	DeleteUtxo(key []byte) error
	GetTokenState(blockID uint) ([]byte, error)
	PutTokenState(blockID uint, data []byte) error
	DeleteTokenState(blockID uint) error
	GetBestChainState() ([]byte, error)
	PutBestChainState(data []byte) error
	PutBlock(block *types.SerializedBlock) error
	HasBlock(hash *hash.Hash) bool
}
