package consensus

import (
	"context"
	"github.com/Qitmeer/qng-core/config"
	"github.com/Qitmeer/qng-core/consensus"
)

type Context struct {
	context.Context
	Cfg *config.Config
	Tp consensus.TxPool
}

func (ctx *Context) GetConfig() *config.Config {
	return ctx.Cfg
}

func (ctx *Context) GetTxPool() consensus.TxPool {
	return ctx.Tp
}