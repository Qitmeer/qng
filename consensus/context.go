package consensus

import (
	"context"
	"github.com/Qitmeer/qng/core/protocol"
)

type Context struct {
	context.Context

	NetworkID protocol.Network
	ChainID   uint32
	NodeID    uint32
	Datadir   string
	LogLevel  string
	LogLocate bool
}
