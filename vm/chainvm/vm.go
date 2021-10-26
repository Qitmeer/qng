package chainvm

import (
	"context"
	"github.com/Qitmeer/qng/vm/chainvm/proto"
	"github.com/Qitmeer/qng/vm/common"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "VM_PLUGIN",
	MagicCookieValue: "dynamic",
}

var PluginMap = map[string]plugin.Plugin{
	"vm": &Plugin{},
}

type Plugin struct {
	plugin.NetRPCUnsupportedPlugin
	vm common.ChainVM
}

func New(vm common.ChainVM) *Plugin { return &Plugin{vm: vm} }

// GRPCServer registers a new GRPC server.
func (p *Plugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterVMServer(s, NewServer(p.vm, broker))
	return nil
}

// GRPCClient returns a new GRPC client
func (p *Plugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return NewVMClient(proto.NewVMClient(c), broker), nil
}
