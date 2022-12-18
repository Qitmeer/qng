package consensus

import (
	"context"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
)

type Context interface {
	context.Context
	GetConfig() *config.Config
	GetTxPool() model.TxPool
	GetNotify() Notify
}
