package meer

type PublicMeerChainAPI struct {
	mc *MeerChain
}

func NewPublicMeerChainAPI(mc *MeerChain) *PublicMeerChainAPI {
	return &PublicMeerChainAPI{mc}
}

func (api *PublicMeerChainAPI) GetMeerChainInfo() (interface{}, error) {
	mi := MeerChainInfo{
		MeerVer:   Version,
		EvmVer:    api.mc.chain.Config().Node.Version,
		ChainID:   api.mc.chain.Config().Eth.Genesis.Config.ChainID.Uint64(),
		NetworkID: api.mc.chain.Config().Eth.NetworkId,
	}
	if len(api.mc.chain.Config().Node.IPCEndpoint()) > 0 {
		mi.IPC = api.mc.chain.Config().Node.IPCEndpoint()
	}
	if len(api.mc.chain.Config().Node.HTTPHost) > 0 {
		mi.HTTP = api.mc.chain.Config().Node.HTTPEndpoint()
	}
	if len(api.mc.chain.Config().Node.WSHost) > 0 {
		mi.WS = api.mc.chain.Config().Node.WSEndpoint()
	}
	return mi, nil
}

type MeerChainInfo struct {
	MeerVer   string `json:"meerver"`
	EvmVer    string `json:"evmver"`
	ChainID   uint64 `json:"chainid"`
	NetworkID uint64 `json:"networkid"`
	IPC       string `json:"ipc,omitempty"`
	HTTP      string `json:"http,omitempty"`
	WS        string `json:"ws,omitempty"`
}
