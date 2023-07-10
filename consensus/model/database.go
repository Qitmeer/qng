package model

type DataBase interface {
	Name() string
	Close()
}
