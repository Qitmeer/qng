package chainvm

import (
	"context"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/vm/chainvm/proto"

	"github.com/hashicorp/go-plugin"
)

type VMServer struct {
	proto.UnimplementedVMServer
	vm     consensus.ChainVM
	broker *plugin.GRPCBroker

	ctx    *context.Context
	closed chan struct{}
}

func NewServer(vm consensus.ChainVM, broker *plugin.GRPCBroker) *VMServer {
	return &VMServer{
		vm:     vm,
		broker: broker,
	}
}
