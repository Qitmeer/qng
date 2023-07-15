package model

import "github.com/Qitmeer/qng/database/common"

type DataBase interface {
	Name() string
	Close()
	Rebuild(mgr IndexManager) error
	GetInfo() (*common.DatabaseInfo, error)
	PutInfo(di *common.DatabaseInfo) error
}
