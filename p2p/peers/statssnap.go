package peers

import (
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/core/protocol"
	"github.com/Qitmeer/qng/v2/meerdag"
	"github.com/Qitmeer/qng/v2/p2p/common"
	"github.com/Qitmeer/qng/v2/params"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

// StatsSnap is a snapshot of peer stats at a point in time.
type StatsSnap struct {
	NodeID         string
	PeerID         peer.ID
	QNR            string
	Address        string
	Protocol       uint32
	Genesis        *hash.Hash
	Services       protocol.ServiceFlag
	Name           string
	Version        string
	Network        string
	State          bool
	Direction      network.Direction
	GraphState     *meerdag.GraphState
	GraphStateDur  time.Duration
	TimeOffset     int64
	ConnTime       time.Duration
	LastSend       time.Time
	LastRecv       time.Time
	BytesSent      uint64
	BytesRecv      uint64
	IsCircuit      bool
	Bads           []string
	ReConnect      uint64
	StateRoot      string
	MempoolReqTime time.Time
	Tasks          int
	Broadcast      int
	MeerState      *common.MeerState
	InSnapSync     bool
	LongConnStat   bool
}

func (p *StatsSnap) IsRelay() bool {
	return protocol.HasServices(p.Services, protocol.Relay)
}

func (p *StatsSnap) IsSnap() bool {
	return protocol.HasServices(p.Services, protocol.Snap)
}

func (p *StatsSnap) IsTheSameNetwork() bool {
	return params.ActiveNetParams.Name == p.Network
}
