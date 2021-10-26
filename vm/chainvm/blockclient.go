package chainvm

import (
	"context"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/vm/chainvm/proto"
	"github.com/Qitmeer/qng/vm/common"
	"time"
)

type BlockClient struct {
	vm *VMClient

	// TODO:
	id       *hash.Hash
	parentID *hash.Hash
	status   common.Status
	bytes    []byte
	height   uint64
	time     time.Time
}

func (b *BlockClient) ID() *hash.Hash { return b.id }

func (b *BlockClient) Accept() error {
	b.status = common.Accepted
	_, err := b.vm.client.BlockAccept(context.Background(), &proto.BlockAcceptRequest{
		Id: b.id[:],
	})
	return err
}

func (b *BlockClient) Reject() error {
	b.status = common.Rejected
	_, err := b.vm.client.BlockReject(context.Background(), &proto.BlockRejectRequest{
		Id: b.id[:],
	})
	return err
}

func (b *BlockClient) Status() common.Status { return b.status }

func (b *BlockClient) Parent() *hash.Hash {
	return b.parentID
}

func (b *BlockClient) Verify() error {
	resp, err := b.vm.client.BlockVerify(context.Background(), &proto.BlockVerifyRequest{
		Bytes: b.bytes,
	})
	if err != nil {
		return err
	}
	return b.time.UnmarshalBinary(resp.Timestamp)
}

func (b *BlockClient) Bytes() []byte        { return b.bytes }
func (b *BlockClient) Height() uint64       { return b.height }
func (b *BlockClient) Timestamp() time.Time { return b.time }
