package vm

import (
	"context"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/vm/consensus"
)

type Context struct {
	context.Context
	Cfg       *config.Config
	Tp        model.TxPool
	Notify    consensus.Notify
	Consensus model.Consensus
}

func (ctx *Context) GetConfig() *config.Config {
	return ctx.Cfg
}

func (ctx *Context) GetTxPool() model.TxPool {
	return ctx.Tp
}

func (ctx *Context) GetNotify() consensus.Notify {
	return ctx.Notify
}

func (ctx Context) GetConsensus() model.Consensus {
	return ctx.Consensus
}
