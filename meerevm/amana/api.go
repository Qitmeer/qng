package amana

import (
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"strings"
)

type PublicAmanaServiceAPI struct {
	q *AmanaService
}

func NewPublicAmanaServiceAPI(q *AmanaService) *PublicAmanaServiceAPI {
	return &PublicAmanaServiceAPI{q}
}

func (api *PublicAmanaServiceAPI) GetAmanaNodeInfo() (interface{}, error) {
	ni := api.q.chain.Node().Server().NodeInfo()

	qi := AmanaInfo{
		ID:         ni.ID,
		Name:       ni.Name,
		Enode:      ni.Enode,
		ENR:        ni.ENR,
		IP:         ni.IP,
		Ports:      ni.Ports,
		ListenAddr: ni.ListenAddr,
		ChainID:    api.q.chain.Config().Eth.Genesis.Config.ChainID.Uint64(),
		NetworkID:  api.q.chain.Config().Eth.NetworkId,
	}
	if len(api.q.chain.Config().Node.IPCEndpoint()) > 0 {
		qi.IPC = api.q.chain.Config().Node.IPCEndpoint()
	}
	if len(api.q.chain.Config().Node.HTTPHost) > 0 {
		qi.HTTP = api.q.chain.Config().Node.HTTPEndpoint()
	}
	if len(api.q.chain.Config().Node.WSHost) > 0 {
		qi.WS = api.q.chain.Config().Node.WSEndpoint()
	}
	return qi, nil
}

type AmanaInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Enode string `json:"enode"`
	ENR   string `json:"enr"`
	IP    string `json:"ip"`
	Ports struct {
		Discovery int `json:"discovery"`
		Listener  int `json:"listener"`
	} `json:"ports"`
	ListenAddr string `json:"listenAddr"`
	ChainID    uint64 `json:"chainid"`
	NetworkID  uint64 `json:"networkid"`
	IPC        string `json:"ipc,omitempty"`
	HTTP       string `json:"http,omitempty"`
	WS         string `json:"ws,omitempty"`
}

func (api *PublicAmanaServiceAPI) GetAmanaPeerInfo() (interface{}, error) {
	pis := api.q.chain.Node().Server().PeersInfo()
	retM := map[string]struct{}{}
	ret := []*p2p.PeerInfo{}
	for _, pi := range pis {
		_, ok := retM[pi.ID]
		if ok {
			continue
		}
		has := false
		if strings.HasPrefix(pi.Name, "amana") {
			has = true
		} else if len(pi.ENR) > 0 {
			node, err := enode.Parse(enode.ValidSchemes, pi.ENR)
			if err != nil {
				continue
			}
			filter := forkid.NewStaticFilter(api.q.chain.Config().Eth.Genesis.Config, api.q.chain.Config().Eth.Genesis.ToBlock().Hash())

			var eth struct {
				ForkID forkid.ID
				Tail   []rlp.RawValue `rlp:"tail"`
			}
			err = node.Load(enr.WithEntry("eth", &eth))
			if err != nil {
				continue
			}

			err = filter(eth.ForkID)
			if err == nil {
				has = true
			}
		}

		if has {
			retM[pi.ID] = struct{}{}
			ret = append(ret, pi)
		}
	}

	return ret, nil
}
