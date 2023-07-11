package model

type DataBase interface {
	Name() string
	Close()
	Rebuild(mgr IndexManager) error
}
