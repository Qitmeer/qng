package consensus

import (
	"context"
	"github.com/Qitmeer/qng/config"
)

type Context interface {
	context.Context
	GetConfig() *config.Config
	GetTxPool() TxPool
	GetNotify() Notify
}
