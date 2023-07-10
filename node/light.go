// Copyright (c) 2017-2018 The qitmeer developers
package node

import (
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/node/service"
)

// QitmeerLight implements the qitmeer light node service.
type QitmeerLight struct {
	service.Service
	// database
	db     legacydb.DB
	config *config.Config
}

func newQitmeerLight(n *Node) (*QitmeerLight, error) {
	light := QitmeerLight{
		config: n.Config,
		db:     n.DB,
	}
	return &light, nil
}
