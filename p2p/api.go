package p2p

import (
	"fmt"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/libp2p/go-libp2p-core/peer"
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
func (api *PublicP2PAPI) GetPeerInfo(verbose *bool, network *string) (interface{}, error) {
	vb := false
	if verbose != nil {
		vb = *verbose
	}
	networkName := ""
	if network != nil {
		networkName = *network
	}
	if len(networkName) <= 0 {
		networkName = params.ActiveNetParams.Name
	}
	ps := api.s
	peers := ps.Peers().StatsSnapshots()
	infos := make([]*json.GetPeerInfoResult, 0, len(peers))
	for _, p := range peers {

		if len(networkName) != 0 && networkName != "all" {
			if p.Network != networkName {
				continue
			}
		}

		if !vb {
			if !p.State.IsConnected() {
				continue
			}
		}
		info := &json.GetPeerInfoResult{
			ID:        p.PeerID,
			Name:      p.Name,
			Address:   p.Address,
			BytesSent: p.BytesSent,
			BytesRecv: p.BytesRecv,
			Circuit:   p.IsCircuit,
			Bads:      p.Bads,
		}
		info.Protocol = p.Protocol
		info.Services = p.Services.String()
		if p.Genesis != nil {
			info.Genesis = p.Genesis.String()
		}
		if p.IsTheSameNetwork() {
			info.State = p.State.String()
		}
		if len(p.Version) > 0 {
			info.Version = p.Version
		}
		if len(p.Network) > 0 {
			info.Network = p.Network
		}

		if p.State.IsConnected() {
			info.TimeOffset = p.TimeOffset
			if p.Genesis != nil {
				info.Genesis = p.Genesis.String()
			}
			info.Direction = p.Direction.String()
			if p.GraphState != nil {
				info.GraphState = marshal.GetGraphStateResult(p.GraphState)
			}
			if ps.PeerSync().SyncPeer() != nil {
				info.SyncNode = p.PeerID == ps.PeerSync().SyncPeer().GetID().String()
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
		if len(p.QNR) > 0 {
			info.QNR = p.QNR
		}
		infos = append(infos, info)
	}
	return infos, nil
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
	api.s.PeerSync().Disconnect(pe)
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
