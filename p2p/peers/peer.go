package peers

import (
	"fmt"
	"github.com/Qitmeer/qng/common/bloom"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/common"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/p2p/qnode"
	"github.com/Qitmeer/qng/p2p/qnr"
	"github.com/Qitmeer/qng/params"
	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"sync"
	"time"
)

var (
	// maxBadResponses is the maximum number of bad responses from a peer before we stop talking to it.
	MaxBadResponses = 50
)

const (
	MinBroadcastRecord  = 10
	BroadcastRecordLife = 30 * time.Minute
	BadResponseLife     = 30 * time.Second
)

// Peer represents a connected p2p network remote node.
type Peer struct {
	*peerStatus
	pid       peer.ID
	syncPoint *hash.Hash
	// Use to fee filter
	feeFilter int64
	filter    *bloom.Filter

	lock       *sync.RWMutex
	lastSend   time.Time
	lastRecv   time.Time
	bytesSent  uint64
	bytesRecv  uint64
	conTime    time.Time
	timeOffset int64

	bidChanCap time.Time

	HSlock         *sync.RWMutex
	graphStateTime time.Time

	rateTasks map[string]*time.Timer

	broadcast map[string]interface{}

	reconnect uint64

	mempoolreq time.Time
}

func (p *Peer) GetID() peer.ID {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.pid
}

// IDWithAddress returns the printable id and address of the remote peer.
// It's useful on printing out the trace log messages.
func (p *Peer) IDWithAddress() string {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.idWithAddress()
}

func (p *Peer) idWithAddress() string {
	return fmt.Sprintf("%s %s", p.pid, p.address)
}

// BadResponses obtains the number of bad responses we have received from the given remote peer.
// This will error if the peer does not exist.
func (p *Peer) BadResponses() []*BadResponse {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.badResponses
}

func (p *Peer) badResponseStrs() []string {
	brs := []string{}
	for _, v := range p.badResponses {
		brs = append(brs, v.String())
	}
	return brs
}

// IsBad states if the peer is to be considered bad.
// If the peer is unknown this will return `false`, which makes using this function easier than returning an error.
func (p *Peer) IsBad() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isBad()
}

func (p *Peer) isBad() bool {
	l := len(p.badResponses)
	if l <= 0 {
		return false
	}
	return time.Since(p.badResponses[l-1].Time) <= BadResponseLife
}

// IncrementBadResponses increments the number of bad responses we have received from the given remote peer.
func (p *Peer) IncrementBadResponses(err *common.Error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.badResponses == nil {
		p.badResponses = []*BadResponse{}
	}
	l := len(p.badResponses)
	br := &BadResponse{Time: time.Now(), Err: err}
	if l <= 0 {
		br.ID = 1
	} else {
		br.ID = p.badResponses[l-1].ID + 1
	}
	if l > 0 && p.badResponses[l-1].Err.String() == err.String() {
		p.badResponses[l-1] = br
	} else {
		p.badResponses = append(p.badResponses, br)
	}

	log.Debug("Bad responses", "peer", p.idWithAddress(), "err", err.String())
	//
	if len(p.badResponses) > MaxBadResponses {
		p.badResponses = p.badResponses[1:]
	}
}

func (p *Peer) ResetBad() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.badResponses = []*BadResponse{}

	log.Debug(fmt.Sprintf("Bad responses reset:%s", p.pid.String()))
}

func (p *Peer) UpdateAddrDir(record *qnr.Record, address ma.Multiaddr, direction network.Direction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.address = address
	p.direction = direction
	if record != nil {
		p.qnr = record
	}
}

// Address returns the multiaddress of the given remote peer.
// This will error if the peer does not exist.
func (p *Peer) Address() ma.Multiaddr {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.address
}

func (p *Peer) QAddress() *common.QMultiaddr {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.qaddress()
}

func (p *Peer) qaddress() *common.QMultiaddr {
	if p.address == nil {
		return nil
	}
	qma, err := common.QMultiAddrFromString(fmt.Sprintf("%s", p.address.String()+"/p2p/"+p.pid.String()))
	if err != nil {
		return nil
	}
	return qma
}

// Direction returns the direction of the given remote peer.
// This will error if the peer does not exist.
func (p *Peer) Direction() network.Direction {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.direction
}

// QNR returns the enr for the corresponding peer id.
func (p *Peer) QNR() *qnr.Record {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.qnr
}

func (p *Peer) Node() *qnode.Node {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.node()
}

func (p *Peer) Filter() *bloom.Filter {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.filter
}

func (p *Peer) node() *qnode.Node {
	if p.qnr == nil {
		return nil
	}

	n, err := qnode.New(qnode.ValidSchemes, p.qnr)
	if err != nil {
		log.Error("qnode: can't verify local record: %v", err)
		return nil
	}
	return n
}

// ConnectionState gets the connection state of the given remote peer.
// This will error if the peer does not exist.
func (p *Peer) ConnectionState() PeerConnectionState {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.peerState
}

// IsActive checks if a peers is active and returns the result appropriately.
func (p *Peer) IsActive() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.isBad() {
		return false
	}
	if !p.canConnectWithNetwork() {
		return false
	}
	return p.peerState.IsConnected()
}

func (p *Peer) IsConnected() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.peerState.IsConnected()
}

// SetConnectionState sets the connection state of the given remote peer.
func (p *Peer) SetConnectionState(state PeerConnectionState) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.peerState = state

	if state.IsConnected() || state.IsDisconnected() {
		p.conTime = time.Now()
	}
}

// SetChainState sets the chain state of the given remote peer.
func (p *Peer) SetChainState(chainState *pb.ChainState) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.chainState = chainState
	p.chainStateLastUpdated = time.Now()
	p.timeOffset = int64(p.chainState.Timestamp) - roughtime.Now().Unix()
	p.graphStateTime = time.Now()
	p.stateRootOrder = uint64(chainState.GraphState.MainOrder)
	log.Trace(fmt.Sprintf("SetChainState(%s) : MainHeight=%d", p.pid.ShortString(), chainState.GraphState.MainHeight))
}

// ChainState gets the chain state of the given remote peer.
// This can return nil if there is no known chain state for the peer.
// This will error if the peer does not exist.
func (p *Peer) ChainState() *pb.ChainState {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.chainState
}

// ChainStateLastUpdated gets the last time the chain state of the given remote peer was updated.
// This will error if the peer does not exist.
func (p *Peer) ChainStateLastUpdated() time.Time {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.chainStateLastUpdated
}

// SetMetadata sets the metadata of the given remote peer.
func (p *Peer) SetMetadata(metaData *pb.MetaData) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.metaData = metaData
}

// Metadata returns a copy of the metadata corresponding to the provided
// peer id.
func (p *Peer) Metadata() *pb.MetaData {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return proto.Clone(p.metaData).(*pb.MetaData)
}

// CommitteeIndices retrieves the committee subnets the peer is subscribed to.
func (p *Peer) CommitteeIndices() []uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.qnr == nil || p.metaData == nil {
		return []uint64{}
	}
	return retrieveIndicesFromBitfield(p.metaData.Subnets)
}

func (p *Peer) StatsSnapshot() (*StatsSnap, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	ss := &StatsSnap{
		PeerID:         p.pid,
		Protocol:       p.protocolVersion(),
		Genesis:        p.genesis(),
		Services:       p.services(),
		Name:           p.getName(),
		Version:        p.getVersion(),
		Network:        p.getNetwork(),
		State:          p.peerState.IsConnected(),
		Direction:      p.direction,
		TimeOffset:     p.timeOffset,
		ConnTime:       time.Since(p.conTime),
		LastSend:       p.lastSend,
		LastRecv:       p.lastRecv,
		BytesSent:      p.bytesSent,
		BytesRecv:      p.bytesRecv,
		IsCircuit:      p.isCircuit(),
		Bads:           p.badResponseStrs(),
		ReConnect:      p.reconnect,
		StateRoot:      p.stateRootAndOrder(),
		MempoolReqTime: p.mempoolreq,
	}
	n := p.node()
	if n != nil {
		ss.NodeID = n.ID().String()
		ss.QNR = n.String()
	}
	if p.qaddress() != nil {
		ss.Address = p.qaddress().String()
	}
	if p.isConsensus() {
		ss.GraphState = p.graphState()
		ss.GraphStateDur = time.Since(p.graphStateTime)
	}
	return ss, nil
}

func (p *Peer) Timestamp() time.Time {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.chainState == nil {
		return time.Time{}
	}
	return time.Unix(int64(p.chainState.Timestamp), 0)
}

func (p *Peer) SetQNR(record *qnr.Record) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.qnr = record
}

func (p *Peer) protocolVersion() uint32 {
	if p.chainState == nil {
		return 0
	}
	return p.chainState.ProtocolVersion
}

func (p *Peer) genesis() *hash.Hash {
	if p.chainState == nil {
		return nil
	}
	if p.chainState.GenesisHash == nil {
		return nil
	}
	genesisHash, err := hash.NewHash(p.chainState.GenesisHash.Hash)
	if err != nil {
		return nil
	}
	return genesisHash
}

func (p *Peer) SetStateRoot(stateRoot *hash.Hash, order uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.stateRoot = stateRoot
	p.stateRootOrder = order
}

func (p *Peer) stateRootAndOrder() string {
	if p.stateRoot == nil {
		return ""
	}
	return fmt.Sprintf("%s (order:%d)", p.stateRoot.String(), p.stateRootOrder)
}

func (p *Peer) Services() protocol.ServiceFlag {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.services()
}

func (p *Peer) services() protocol.ServiceFlag {
	if p.chainState == nil {
		return protocol.Unknown
	}
	return protocol.ServiceFlag(p.chainState.Services)
}

func (p *Peer) userAgent() string {
	if p.chainState == nil {
		return ""
	}
	return string(p.chainState.UserAgent)
}

func (p *Peer) graphState() *meerdag.GraphState {
	if p.chainState == nil {
		return nil
	}
	gs := meerdag.NewGraphState()
	gs.SetTotal(uint(p.chainState.GraphState.Total))
	gs.SetLayer(uint(p.chainState.GraphState.Layer))
	gs.SetMainHeight(uint(p.chainState.GraphState.MainHeight))
	gs.SetMainOrder(uint(p.chainState.GraphState.MainOrder))
	tips := []*hash.Hash{}
	for _, tip := range p.chainState.GraphState.Tips {
		h, err := hash.NewHash(tip.Hash)
		if err != nil {
			return nil
		}
		tips = append(tips, h)
	}
	gs.SetTips(tips)
	return gs
}

func (p *Peer) GraphState() *meerdag.GraphState {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.graphState()
}

func (p *Peer) UpdateGraphState(gs *pb.GraphState) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.chainState == nil {
		p.chainState = &pb.ChainState{}
		//per.chainState.GraphState=&pb.GraphState{}
	}
	p.chainState.GraphState = gs
	log.Trace(fmt.Sprintf("UpdateGraphState(%s) : MainHeight=%d", p.pid.ShortString(), gs.MainHeight))

	p.graphStateTime = time.Now()
	/*	per.chainState.GraphState.Total=uint32(gs.GetTotal())
		per.chainState.GraphState.Layer=uint32(gs.GetLayer())
		per.chainState.GraphState.MainOrder=uint32(gs.GetMainOrder())
		per.chainState.GraphState.MainHeight=uint32(gs.GetMainHeight())
		per.chainState.GraphState.Tips=[]*pb.Hash{}
		for h:=range gs.GetTips().GetMap() {
			per.chainState.GraphState.Tips=append(per.chainState.GraphState.Tips,&pb.Hash{Hash:h.Bytes()})
		}*/
}

func (p *Peer) UpdateSyncPoint(point *hash.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.syncPoint = point
}

func (p *Peer) SyncPoint() *hash.Hash {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.syncPoint
}

func (p *Peer) DisableRelayTx() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.chainState == nil {
		return false
	}
	return p.chainState.DisableRelayTx
}

func (p *Peer) FeeFilter() int64 {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.feeFilter
}

func (p *Peer) ConnectionTime() time.Time {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.conTime
}

func (p *Peer) IsRelay() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.chainState == nil {
		return false
	}
	return protocol.HasServices(protocol.ServiceFlag(p.chainState.Services), protocol.Relay)
}

func (p *Peer) IsConsensus() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.isConsensus()
}

func (p *Peer) isConsensus() bool {
	if p.chainState == nil {
		return false
	}
	return HasConsensusService(protocol.ServiceFlag(p.chainState.Services))
}

func (p *Peer) IncreaseBytesSent(size int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.bytesSent += uint64(size)
	p.lastSend = time.Now()
}

func (p *Peer) BytesSent() uint64 {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.bytesSent
}

func (p *Peer) IncreaseBytesRecv(size int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.bytesRecv += uint64(size)
	p.lastRecv = time.Now()
}

func (p *Peer) BytesRecv() uint64 {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.bytesRecv
}

func (p *Peer) GetName() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.getName()
}

func (p *Peer) getName() string {
	err, name, _, _ := ParseUserAgent(p.userAgent())
	if err != nil {
		return p.userAgent()
	}
	return name
}

func (p *Peer) GetVersion() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.getVersion()
}

func (p *Peer) getVersion() string {

	err, _, version, _ := ParseUserAgent(p.userAgent())
	if err != nil {
		return ""
	}
	return version
}

func (p *Peer) GetNetwork() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.getNetwork()
}

func (p *Peer) getNetwork() string {
	err, _, _, network := ParseUserAgent(p.userAgent())
	if err != nil {
		return ""
	}
	return network
}

func (p *Peer) CanConnectWithNetwork() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.canConnectWithNetwork()
}

func (p *Peer) canConnectWithNetwork() bool {
	network := p.getNetwork()
	if len(network) <= 0 {
		return true
	}
	return params.ActiveNetParams.Name == network
}

func (p *Peer) GetBidChanCap() time.Time {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.bidChanCap
}

func (p *Peer) SetBidChanCap(life time.Time) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.bidChanCap = life
}

func (p *Peer) isCircuit() bool {
	if p.direction == network.DirOutbound {
		return true
	}
	return !p.bidChanCap.IsZero()
}

func (p *Peer) RunRate(task string, delay time.Duration, f func()) {
	p.lock.Lock()
	defer p.lock.Unlock()

	rt, ok := p.rateTasks[task]
	if !ok {
		rt = time.NewTimer(delay)
		p.rateTasks[task] = rt
		go func() {
			select {
			case <-rt.C:
				f()
			}
			p.lock.Lock()
			delete(p.rateTasks, task)
			p.lock.Unlock()
		}()

		return
	}
	rt.Reset(delay)
}

func (p *Peer) Broadcast(key string, record interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.broadcast[key] = record
}

func (p *Peer) HasBroadcast(key string) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	_, ok := p.broadcast[key]
	return ok
}

func (p *Peer) UpdateBroadcast() {
	p.lock.Lock()
	defer p.lock.Unlock()

	for key, data := range p.broadcast {
		switch value := data.(type) {
		case *types.TxDesc:
			if time.Since(value.Added) > BroadcastRecordLife && len(p.broadcast) > MinBroadcastRecord {
				delete(p.broadcast, key)
			}
		}
	}
}

func (p *Peer) IncreaseReConnect() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.reconnect++
}

func (p *Peer) GetMempoolReqTime() time.Time {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.mempoolreq
}

func (p *Peer) SetMempoolReqTime(t time.Time) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.mempoolreq = t
}

func NewPeer(pid peer.ID, point *hash.Hash) *Peer {
	return &Peer{
		peerStatus: &peerStatus{
			peerState: PeerDisconnected,
		},
		pid:       pid,
		lock:      &sync.RWMutex{},
		HSlock:    &sync.RWMutex{},
		syncPoint: point,
		filter:    bloom.LoadFilter(nil),
		rateTasks: map[string]*time.Timer{},
		broadcast: map[string]interface{}{},
	}
}
