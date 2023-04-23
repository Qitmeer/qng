package peers

import (
	"errors"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/qnr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"sync"
)

var (
	// ErrPeerUnknown is returned when there is an attempt to obtain data from a peer that is not known.
	ErrPeerUnknown = errors.New("peer unknown")
)

// Status is the structure holding the peer status information.
type Status struct {
	lock  sync.RWMutex
	peers map[peer.ID]*Peer

	p2p P2PRPC
}

// Bad returns the peers that are bad.
func (p *Status) Bad() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()

	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		if status.IsBad() {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Connecting returns the peers that are connecting.
func (p *Status) Connecting() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		if status.ConnectionState().IsConnecting() {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Connected returns the peers that are connected.
func (p *Status) Connected() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		if status.ConnectionState().IsConnected() {
			peers = append(peers, pid)
		}
	}
	return peers
}

func (p *Status) ConnectedPeers() []*Peer {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]*Peer, 0)
	for _, status := range p.peers {
		if status.ConnectionState().IsConnected() {
			peers = append(peers, status)
		}
	}
	return peers
}

func (p *Status) CanSyncPeers() []*Peer {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]*Peer, 0)
	for _, pe := range p.peers {
		if !p.IsActive(pe) ||
			!pe.IsConsensus() {
			continue
		}
		peers = append(peers, pe)
	}
	return peers
}

func (p *Status) AllPeers() []*Peer {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]*Peer, 0)
	for _, status := range p.peers {
		peers = append(peers, status)
	}
	return peers
}

// Active returns the peers that are connecting or connected.
func (p *Status) Active() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, pe := range p.peers {
		if p.IsActive(pe) {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnecting returns the peers that are disconnecting.
func (p *Status) Disconnecting() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		if status.ConnectionState().IsDisconnecting() {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Disconnected returns the peers that are disconnected.
func (p *Status) Disconnected() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		if status.ConnectionState().IsDisconnected() {
			peers = append(peers, pid)
		}
	}
	return peers
}

// Inactive returns the peers that are disconnecting or disconnected.
func (p *Status) Inactive() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, pe := range p.peers {
		if !p.IsActive(pe) {
			peers = append(peers, pid)
		}
	}
	return peers
}

// All returns all the peers regardless of state.
func (p *Status) All() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pids := make([]peer.ID, 0, len(p.peers))
	for pid := range p.peers {
		pids = append(pids, pid)
	}
	return pids
}

func (p *Status) DirInbound() []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()
	peers := make([]peer.ID, 0)
	for pid, pe := range p.peers {
		if p.IsActive(pe) && pe.Direction() == network.DirInbound {
			peers = append(peers, pid)
		}
	}
	return peers
}

// fetch is a helper function that fetches a peer, possibly creating it.
func (p *Status) Get(pid peer.ID) *Peer {
	p.lock.RLock()
	defer p.lock.RUnlock()

	pe, ok := p.peers[pid]
	if !ok {
		return nil
	}
	return pe
}

// fetch is a helper function that fetches a peer, possibly creating it.
func (p *Status) Fetch(pid peer.ID) *Peer {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, ok := p.peers[pid]; !ok {
		var genHash *hash.Hash
		if p.p2p != nil {
			genHash = p.p2p.GetGenesisHash()
		}
		p.peers[pid] = NewPeer(pid, genHash)
	}
	return p.peers[pid]
}

// Add adds a peer.
// If a peer already exists with this ID its address and direction are updated with the supplied data.
func (p *Status) Add(record *qnr.Record, pid peer.ID, address ma.Multiaddr, direction network.Direction) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if pe, ok := p.peers[pid]; ok {
		// Peer already exists, just update its address info.
		pe.UpdateAddrDir(record, address, direction)
		return
	}
	pe := NewPeer(pid, p.p2p.GetGenesisHash())
	pe.UpdateAddrDir(record, address, direction)

	p.peers[pid] = pe
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
func (p *Status) IncrementBadResponses(pid peer.ID, err *common.Error) {
	pe := p.Get(pid)
	if pe == nil {
		return
	}
	pe.IncrementBadResponses(err)
}

// SubscribedToSubnet retrieves the peers subscribed to the given
// committee subnet.
func (p *Status) SubscribedToSubnet(index uint64) []peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()

	peers := make([]peer.ID, 0)
	for pid, status := range p.peers {
		// look at active peers
		if p.IsActive(status) && status.metaData != nil && status.metaData.Subnets != nil {
			indices := retrieveIndicesFromBitfield(status.metaData.Subnets)
			for _, idx := range indices {
				if idx == index {
					peers = append(peers, pid)
					break
				}
			}
		}
	}
	return peers
}

func (p *Status) StatsSnapshots() []*StatsSnap {
	p.lock.RLock()
	defer p.lock.RUnlock()

	pes := make([]*StatsSnap, 0, len(p.peers))
	for _, pe := range p.peers {
		ss, err := pe.StatsSnapshot()
		if err != nil {
			continue
		}
		pes = append(pes, ss)
	}
	return pes
}

func (p *Status) ForPeers(state PeerConnectionState, closure func(pe *Peer)) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, pe := range p.peers {
		if pe.ConnectionState() != state {
			continue
		}
		closure(pe)
	}
}

func (p *Status) UpdateBroadcasts() {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, pe := range p.peers {
		pe.UpdateBroadcast()
	}
}

func (p *Status) CanConnect(pid peer.ID) bool {
	cs := p.p2p.Host().Network().Connectedness(pid)
	return cs == network.CanConnect || cs == network.Connected
}

func (p *Status) IsActive(pe *Peer) bool {
	return pe.IsActive() && p.CanConnect(pe.GetID())
}

func (p *Status) IsActiveID(pid peer.ID) bool {
	pe := p.Get(pid)
	if pe == nil {
		return false
	}
	return p.IsActive(pe)
}

func (p *Status) GetByAddress(address ma.Multiaddr) *Peer {
	p.lock.RLock()
	defer p.lock.RUnlock()
	if address == nil {
		return nil
	}
	for _, pe := range p.peers {
		addr := pe.Address()
		if addr == nil {
			continue
		}
		if addr.Equal(address) {
			return pe
		}
	}
	return nil
}

// NewStatus creates a new status entity.
func NewStatus(p2p P2PRPC) *Status {
	return &Status{
		p2p:   p2p,
		peers: make(map[peer.ID]*Peer),
	}
}
