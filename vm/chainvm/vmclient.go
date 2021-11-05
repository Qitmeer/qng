/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package chainvm

import (
	"context"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus"
	"github.com/Qitmeer/qng/vm/chainvm/proto"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type VMClient struct {
	*consensus.ChainState
	client proto.VMClient
	broker *plugin.GRPCBroker
	proc   *plugin.Client

	conns []*grpc.ClientConn

	ctx context.Context
}

func (vm *VMClient) SetProcess(proc *plugin.Client) {
	vm.proc = proc
}

func (vm *VMClient) Initialize(ctx *consensus.Context) error {
	vm.ctx = ctx

	resp, err := vm.client.Initialize(ctx, &proto.InitializeRequest{
		NetworkID: uint32(ctx.NetworkID), ChainID: ctx.ChainID, NodeID: ctx.NodeID, Datadir: ctx.Datadir, LogLevel: ctx.LogLevel, LogLocate: ctx.LogLocate})
	if err != nil {
		return err
	}
	id, err := hash.NewHash(resp.LastAcceptedID)
	if err != nil {
		return err
	}
	parentID, err := hash.NewHash(resp.LastAcceptedParentID)
	if err != nil {
		return err
	}
	status := consensus.Status(resp.Status)
	if err := status.Valid(); err != nil {
		log.Error(err.Error())
	}

	timestamp := time.Unix(int64(resp.Timestamp), 0)

	lastAcceptedBlk := &BlockClient{
		vm:       vm,
		id:       id,
		parentID: parentID,
		status:   status,
		bytes:    resp.Bytes,
		height:   resp.Height,
		time:     timestamp,
	}

	vm.ChainState = &consensus.ChainState{LastAcceptedBlock: lastAcceptedBlk}

	return nil
}

func (vm *VMClient) Bootstrapping() error {
	_, err := vm.client.Bootstrapping(context.Background(), &emptypb.Empty{})
	return err
}

func (vm *VMClient) Bootstrapped() error {
	_, err := vm.client.Bootstrapped(context.Background(), &emptypb.Empty{})
	return err
}

func (vm *VMClient) Shutdown() error {
	var ret error
	_, err := vm.client.Shutdown(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Error(err.Error())
		ret = err
	}
	for _, conn := range vm.conns {
		err := conn.Close()
		if err != nil {
			log.Error(err.Error())
			ret = err
		}
	}

	vm.proc.Kill()
	return ret
}

func (vm *VMClient) Version() (string, error) {
	resp, err := vm.client.Version(
		context.Background(),
		&emptypb.Empty{},
	)
	if err != nil {
		return "", err
	}
	return resp.Version, nil
}

func (vm *VMClient) BuildBlock(txs []string) (consensus.Block, error) {
	resp, err := vm.client.BuildBlock(vm.ctx, &proto.BuildBlockRequest{
		Txs: txs,
	})
	if err != nil {
		return nil, err
	}
	id := hash.MustBytesToHash(resp.Id)
	parentID := hash.MustBytesToHash(resp.ParentID)
	return &BlockClient{
		vm:       vm,
		id:       &id,
		parentID: &parentID,
		status:   consensus.Accepted,
		bytes:    resp.Bytes,
		height:   resp.Height,
		time:     time.Unix(int64(resp.Timestamp), 0),
	}, nil
}

func (vm *VMClient) GetBlock(*hash.Hash) (consensus.Block, error) {
	return nil, nil
}

func (vm *VMClient) ParseBlock([]byte) (consensus.Block, error) {
	return nil, nil
}

func (vm *VMClient) LastAccepted() (*hash.Hash, error) {
	return nil, nil
}

func NewVMClient(client proto.VMClient, broker *plugin.GRPCBroker) *VMClient {
	return &VMClient{
		client: client,
		broker: broker,
	}
}
