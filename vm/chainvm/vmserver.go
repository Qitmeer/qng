package chainvm

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/vm/chainvm/proto"
	"github.com/hashicorp/go-plugin"
	"google.golang.org/protobuf/types/known/emptypb"
)

type VMServer struct {
	proto.UnimplementedVMServer
	vm     consensus.ChainVM
	broker *plugin.GRPCBroker

	ctx    *consensus.Context
	closed chan struct{}

	network protocol.Network
	chainID uint32
	nodeID  uint32
}

func NewServer(vm consensus.ChainVM, broker *plugin.GRPCBroker) *VMServer {
	return &VMServer{
		vm:     vm,
		broker: broker,
		closed: make(chan struct{}, 1),
	}
}

func (vm *VMServer) Initialize(ctx context.Context, req *proto.InitializeRequest) (*proto.InitializeResponse, error) {
	vm.network = protocol.Network(req.NetworkID)
	vm.chainID = req.ChainID
	vm.nodeID = req.NodeID

	log.Debug(fmt.Sprintf("network:%d chainID:%d nodeID:%d datadir:%s", vm.network.String(), vm.chainID, vm.nodeID, req.Datadir))

	vm.ctx = &consensus.Context{Context: ctx, NetworkID: protocol.Network(req.NetworkID), ChainID: req.ChainID, NodeID: req.NodeID, Datadir: req.Datadir, LogLevel: req.LogLevel}

	if err := vm.vm.Initialize(vm.ctx); err != nil {
		close(vm.closed)
		return nil, err
	}

	lastAccepted, err := vm.vm.LastAccepted()
	if err != nil {
		// Ignore errors closing resources to return the original error
		_ = vm.vm.Shutdown()
		close(vm.closed)
		return nil, err
	}

	blk, err := vm.vm.GetBlock(lastAccepted)
	if err != nil {
		// Ignore errors closing resources to return the original error
		_ = vm.vm.Shutdown()
		close(vm.closed)
		return nil, err
	}
	parentID := blk.Parent()

	return &proto.InitializeResponse{
		LastAcceptedID:       lastAccepted[:],
		LastAcceptedParentID: parentID[:],
		Status:               uint32(consensus.Accepted),
		Height:               blk.Height(),
		Bytes:                blk.Bytes(),
		Timestamp:            uint64(blk.Timestamp().Unix()),
	}, err
}

func (vm *VMServer) Bootstrapping(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, vm.vm.Bootstrapping()
}

func (vm *VMServer) Bootstrapped(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, vm.vm.Bootstrapped()
}

func (vm *VMServer) Shutdown(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	if vm.closed == nil {
		return &emptypb.Empty{}, nil
	}
	err := vm.vm.Shutdown()
	close(vm.closed)
	return &emptypb.Empty{}, err
}

func (vm *VMServer) Version(context.Context, *emptypb.Empty) (*proto.VersionResponse, error) {
	version, err := vm.vm.Version()
	return &proto.VersionResponse{Version: version}, err
}

func (vm *VMServer) BuildBlock(ctx context.Context, req *proto.BuildBlockRequest) (*proto.BuildBlockResponse, error) {
	blk, err := vm.vm.BuildBlock(req.Txs)
	if err != nil {
		return nil, err
	}
	blkID := blk.ID()
	parentID := blk.Parent()
	return &proto.BuildBlockResponse{
		Id:        blkID[:],
		ParentID:  parentID[:],
		Bytes:     blk.Bytes(),
		Height:    blk.Height(),
		Timestamp: uint64(blk.Timestamp().Unix()),
	}, err
}
