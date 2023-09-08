package peers

import (
	"context"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/encoder"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/qnode"
	"github.com/Qitmeer/qng/services/mempool"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type P2P interface {
	GetGenesisHash() *hash.Hash
	BlockChain() *blockchain.BlockChain
	Host() host.Host
	Disconnect(pid peer.ID) error
	Context() context.Context
	Encoding() encoder.NetworkEncoding
	Config() *common.Config
	TxMemPool() *mempool.TxPool
	Metadata() *pb.MetaData
	MetadataSeq() uint64
	TimeSource() model.MedianTimeSource
	Notify() model.Notify
	ConnectTo(node *qnode.Node)
	Resolve(n *qnode.Node) *qnode.Node
	Node() *qnode.Node
	RelayNodeInfo() *peer.AddrInfo
	IncreaseBytesSent(pid peer.ID, size int)
	IncreaseBytesRecv(pid peer.ID, size int)
	ConnectToPeer(qmaddr string, force bool) error
	RegainMempool()
	IsCurrent() bool
	Peers() *Status
	IsRunning() bool
	Consensus() model.Consensus
}

type P2PRPC interface {
	Host() host.Host
	Context() context.Context
	Encoding() encoder.NetworkEncoding
	Disconnect(pid peer.ID) error
	IncreaseBytesSent(pid peer.ID, size int)
	IncreaseBytesRecv(pid peer.ID, size int)
	Peers() *Status
	IsRunning() bool
	GetGenesisHash() *hash.Hash
}
