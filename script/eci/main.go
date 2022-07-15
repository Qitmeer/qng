package main

import (
	"eci/config"
	"eci/qngEci"
	"flag"
	"fmt"
)

var (
	confPath    = flag.String("config", "./config.conf", "./eci --config=./config.conf")
	action      = flag.String("action", "create", "./eci --action=(create|restart|list|delete)")
	containerId = flag.String("containerId", "", "./eci --containerId=123456")
)

const (
	ACTION_CREATE  = "create"
	ACTION_RESTART = "restart"
	ACTION_LIST    = "list"
	ACTION_DELETE  = "delete"
)

func main() {
	flag.Parse()
	conf := config.NewConfig(confPath)
	eciProvider := qngEci.NewEciInstance(conf)
	switch *action {
	case ACTION_CREATE:
		fmt.Println(eciProvider)
		eciProvider.CreateContainer()
	case ACTION_RESTART:
		eciProvider.RestartContainers(*containerId)
	case ACTION_LIST:
		eciProvider.AllContainers("", nil)
	case ACTION_DELETE:
		eciProvider.DeleteContainer(*containerId)
	default:
		fmt.Println("./eci --action=(create|restart|list|delete) --config=./config.conf --containerId=123456")
	}
}
