package vm

import (
	"context"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/vm/consensus"
)

type Context struct {
	context.Context
	Cfg *config.Config
	Tp consensus.TxPool
	Notify consensus.Notify
}

func (ctx *Context) GetConfig() *config.Config {
	return ctx.Cfg
}

func (ctx *Context) GetTxPool() consensus.TxPool {
	return ctx.Tp
}

func (ctx *Context) GetNotify() consensus.Notify {
	return ctx.Notify
}