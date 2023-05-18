package peers

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
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
}

func (p *StatsSnap) IsRelay() bool {
	return protocol.HasServices(protocol.ServiceFlag(p.Services), protocol.Relay)
}

func (p *StatsSnap) IsTheSameNetwork() bool {
	return params.ActiveNetParams.Name == p.Network
}
