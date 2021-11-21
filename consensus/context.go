package consensus

import (
	"context"
	"github.com/Qitmeer/qitmeer/config"
)

type Context struct {
	context.Context
	Cfg *config.Config
}

func (ctx *Context) GetConfig() *config.Config {
	return ctx.Cfg
}
