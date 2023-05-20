package p2p

import (
	"fmt"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/peer"
	"math"
	"strings"
	"time"
)

func (s *Service) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicP2PAPI(s),
			Public:    true,
		},
		{
			NameSpace: cmds.P2PNameSpace,
			Service:   NewPrivateP2PAPI(s),
			Public:    false,
		},
	}
}

type PublicP2PAPI struct {
	s *Service
}

func NewPublicP2PAPI(s *Service) *PublicP2PAPI {
	return &PublicP2PAPI{s}
}

// Return the peer info
func (api *PublicP2PAPI) GetPeerInfo(verbose *bool, pid *string) (interface{}, error) {
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	pidStr := ""
	if pid != nil {
		pidStr = *pid
	}

	ps := api.s
	peers := ps.Peers().StatsSnapshots()
	infos := make([]*json.GetPeerInfoResult, 0, len(peers))
	for _, p := range peers {
		active := ps.Peers().IsActiveID(p.PeerID)
		if !vb {
			if !active {
				continue
			}
		}
		if len(pidStr) > 0 {
			if p.PeerID.String() != pidStr &&
				!strings.Contains(p.PeerID.String(), pidStr) {
				continue
			}
		}
		info := &json.GetPeerInfoResult{
			ID:        p.PeerID.String(),
			Name:      p.Name,
			Address:   p.Address,
			BytesSent: p.BytesSent,
			BytesRecv: p.BytesRecv,
			Circuit:   p.IsCircuit,
			Bads:      p.Bads,
			ReConnect: p.ReConnect,
			Active:    active,
		}
		info.Protocol = p.Protocol
		info.Services = p.Services.String()
		if p.Genesis != nil {
			info.Genesis = p.Genesis.String()
		}
		if len(p.StateRoot) > 0 {
			info.StateRoot = p.StateRoot
		}
		if p.IsTheSameNetwork() {
			info.State = p.State
		}
		if len(p.Version) > 0 {
			info.Version = p.Version
		}
		if len(p.Network) > 0 {
			info.Network = p.Network
		}

		if p.State {
			info.TimeOffset = p.TimeOffset
			if p.Genesis != nil {
				info.Genesis = p.Genesis.String()
			}
			info.Direction = p.Direction.String()
			if p.GraphState != nil {
				info.GraphState = marshal.GetGraphStateResult(p.GraphState)
			}
			if ps.PeerSync().SyncPeer() != nil {
				info.SyncNode = p.PeerID == ps.PeerSync().SyncPeer().GetID()
			} else {
				info.SyncNode = false
			}
			info.ConnTime = p.ConnTime.Truncate(time.Second).String()
			info.GSUpdate = p.GraphStateDur.Truncate(time.Second).String()
		}
		if !p.LastSend.IsZero() {
			info.LastSend = p.LastSend.String()
		}
		if !p.LastRecv.IsZero() {
			info.LastRecv = p.LastRecv.String()
		}
		if !p.MempoolReqTime.IsZero() {
			info.MempoolReqTime = p.MempoolReqTime.String()
		}
		if len(p.QNR) > 0 {
			info.QNR = p.QNR
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Reload All Peers
func (api *PrivateP2PAPI) ReloadPeers() error {
	api.s.connectFromPeerStore()
	return nil
}

// Return IsCurrent
func (api *PublicP2PAPI) IsCurrent() (interface{}, error) {
	return api.s.IsCurrent(), nil
}

func (api *PublicP2PAPI) GetNetworkInfo() (interface{}, error) {
	ps := api.s
	peers := ps.Peers().StatsSnapshots()
	nstat := &json.NetworkStat{MaxConnected: ps.Config().MaxPeers,
		MaxInbound: ps.Config().MaxInbound, Infos: []*json.NetworkInfo{}}
	infos := map[string]*json.NetworkInfo{}
	gsups := map[string][]time.Duration{}

	for _, p := range peers {
		nstat.TotalPeers++

		if p.Services&protocol.Relay > 0 {
			nstat.TotalRelays++
		}
		//
		if len(p.Network) <= 0 {
			continue
		}

		info, ok := infos[p.Network]
		if !ok {
			info = &json.NetworkInfo{Name: p.Network}
			infos[p.Network] = info
			nstat.Infos = append(nstat.Infos, info)

			gsups[p.Network] = []time.Duration{0, 0, math.MaxInt64}
		}
		info.Peers++
		if ps.Peers().IsActiveID(p.PeerID) {
			info.Connecteds++
			nstat.TotalConnected++

			gsups[p.Network][0] = gsups[p.Network][0] + p.GraphStateDur
			if p.GraphStateDur > gsups[p.Network][1] {
				gsups[p.Network][1] = p.GraphStateDur
			}
			if p.GraphStateDur < gsups[p.Network][2] {
				gsups[p.Network][2] = p.GraphStateDur
			}
		}
		if p.Services&protocol.Relay > 0 {
			info.Relays++
		}
	}
	for k, gu := range gsups {
		info, ok := infos[k]
		if !ok {
			continue
		}
		if info.Connecteds > 0 {
			avegs := time.Duration(0)
			if info.Connecteds > 2 {
				avegs = gu[0] - gu[1] - gu[2]
				if avegs < 0 {
					avegs = 0
				}
				cons := info.Connecteds - 2
				avegs = time.Duration(int64(avegs) / int64(cons))

			} else {
				avegs = time.Duration(int64(gu[0]) / int64(info.Connecteds))
			}

			info.AverageGS = avegs.Truncate(time.Second).String()
			info.MaxGS = gu[1].Truncate(time.Second).String()
			info.MinGS = gu[2].Truncate(time.Second).String()
		}
	}
	return nstat, nil
}

type PrivateP2PAPI struct {
	s *Service
}

func NewPrivateP2PAPI(s *Service) *PrivateP2PAPI {
	return &PrivateP2PAPI{s}
}

func (api *PrivateP2PAPI) AddPeer(qmaddr string) (interface{}, error) {
	err := api.s.ConnectToPeer(qmaddr, true)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (api *PrivateP2PAPI) DelPeer(pid string) (interface{}, error) {
	peid, err := peer.Decode(pid)
	if err != nil {
		return false, err
	}

	pe := api.s.Peers().Get(peid)
	if pe == nil {
		return false, fmt.Errorf("No peer:%s", peid.String())
	}
	api.s.PeerSync().TryDisconnect(pe)
	return true, nil
}

func (api *PrivateP2PAPI) Ping(addr string, port uint, protocol string) (interface{}, error) {
	if len(protocol) <= 0 {
		protocol = "tcp"
	}
	if port == 0 {
		port = api.s.cfg.TCPPort
	}
	return verifyConnectivity(addr, port, protocol)
}

func (api *PrivateP2PAPI) Pause() (interface{}, error) {
	return api.s.PeerSync().Pause(), nil
}

func (api *PrivateP2PAPI) ResetPeers() (interface{}, error) {
	for _, pe := range api.s.Peers().AllPeers() {
		if !api.s.Peers().IsActive(pe) {
			continue
		}
		api.s.PeerSync().TryDisconnect(pe)
	}
	<-time.After(time.Second)
	trynum := 0
	for _, pe := range api.s.Peers().AllPeers() {
		qa := pe.QAddress()
		if qa == nil {
			continue
		}
		err := api.s.ConnectToPeer(qa.String(), true)
		if err != nil {
			log.Error(err.Error())
		} else {
			trynum++
		}
	}

	bootstrap := api.s.cfg.BootstrapNodeAddr
	bootstrap = append(bootstrap, api.s.cfg.StaticPeers...)
	for _, qa := range bootstrap {
		err := api.s.ConnectToPeer(qa, true)
		if err != nil {
			log.Error(err.Error())
		} else {
			trynum++
		}
	}
	return trynum, nil
}

func (api *PrivateP2PAPI) SetLibp2pLogLevel(level string) (interface{}, error) {
	if len(level) <= 0 {
		return "DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL", nil
	}
	l, err := golog.LevelFromString(level)
	if err != nil {
		log.Error(err.Error())
		return level, err
	}
	golog.SetAllLoggers(l)
	return level, nil
}

// Banlist
func (api *PrivateP2PAPI) Banlist() (interface{}, error) {
	bl := api.s.GetBanlist()
	bls := []*json.GetBanlistResult{}
	for k, v := range bl {
		bls = append(bls, &json.GetBanlistResult{PeerID: k.String(), Bads: v})
	}
	return bls, nil
}

// RemoveBan
func (api *PrivateP2PAPI) RemoveBan(id *string) (interface{}, error) {
	ho := ""
	if id != nil {
		ho = *id
	}
	api.s.RemoveBan(ho)
	return true, nil
}

func (api *PrivateP2PAPI) CheckConsistency(hashOrNumber string) (interface{}, error) {
	hn, err := protocol.NewHashOrNumber(hashOrNumber)
	if err != nil {
		log.Warn("Will use the default block", "error", err.Error())
		return api.s.sy.CheckConsistency(nil)
	}
	return api.s.sy.CheckConsistency(hn)
}
