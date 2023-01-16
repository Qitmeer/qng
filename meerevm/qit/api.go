package qit

type PublicQitServiceAPI struct {
	q *QitService
}

func NewPublicQitServiceAPI(q *QitService) *PublicQitServiceAPI {
	return &PublicQitServiceAPI{q}
}

func (api *PublicQitServiceAPI) GetQitInfo() (interface{}, error) {
	ni := api.q.chain.Node().Server().NodeInfo()

	qi := QitInfo{
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

type QitInfo struct {
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
	HTTP       string `json:"ws,omitempty"`
	WS         string `json:"ws,omitempty"`
}
