package consensus

import (
	"context"
	"github.com/Qitmeer/qng-core/config"
)

type Context interface {
	context.Context
	GetConfig() *config.Config
}
