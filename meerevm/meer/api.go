package meer

import (
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/v2/meerevm/meer/meerchange"
	"github.com/Qitmeer/qng/v2/params"
	rpcapi "github.com/Qitmeer/qng/v2/rpc/api"
	"github.com/ethereum/go-ethereum/common"
	"math"
)

type MeerChainInfo struct {
	MeerVer     string        `json:"meerver"`
	EvmVer      string        `json:"evmver"`
	ChainID     uint64        `json:"chainid"`
	NetworkID   uint64        `json:"networkid"`
	IPC         string        `json:"ipc,omitempty"`
	HTTP        string        `json:"http,omitempty"`
	WS          string        `json:"ws,omitempty"`
	SysContract []interface{} `json:"syscontracts,omitempty"`
}

type MeerChangeInfo struct {
	Name    string `json:"name"`
	Addr    string `json:"address"`
	Code    string `json:"code"`
	Version int    `json:"version"`
	Fork    string `json:"fork,omitempty"`
}

type PublicBlockChainAPI struct {
	mc *BlockChain
}

func NewPublicBlockChainAPI(mc *BlockChain) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{mc}
}

func (api *PublicBlockChainAPI) GetMeerChainInfo() (interface{}, error) {
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

	mci := &MeerChangeInfo{
		Name:    "MeerChange",
		Addr:    meerchange.ContractAddr.String(),
		Version: meerchange.Version,
	}
	ret, bc := api.mc.CheckMeerChangeDeploy()
	if ret {
		mci.Code = hex.EncodeToString(bc)
	} else {
		mci.Code = "Not yet deployed"
	}
	mi.SysContract = []interface{}{api.mc.DeterministicDeploymentProxy().Info(), mci}

	header := api.mc.CurrentHeader()
	if header != nil {
		forkNumber := int64(math.MaxInt64)
		if params.ActiveNetParams.MeerChangeForkBlock != nil {
			forkNumber = params.ActiveNetParams.MeerChangeForkBlock.Int64()
		}
		mci.Fork = fmt.Sprintf("%d/%d", header.Number.Uint64(), forkNumber)
	}
	return mi, nil
}

func (api *PublicBlockChainAPI) GetMeerChangeAddr() (interface{}, error) {
	return meerchange.ContractAddr.String(), nil
}

func (api *PublicBlockChainAPI) DeployMeerChange(owner common.Address) (interface{}, error) {
	if api.mc.IsMeerChangeDeployed() {
		log.Info("MeerChange has already been deployed, so ignore this operation")
		return nil, nil
	}
	txHash, err := api.mc.DeterministicDeploymentProxy().DeployContract(owner, common.FromHex(meerchange.MeerchangeMetaData.Bin), meerchange.Version, nil, 0)
	if err != nil {
		return nil, err
	}
	return txHash.String(), nil
}

func (api *PublicBlockChainAPI) HasMeerState(hashOrNumber string) (interface{}, error) {
	hn, err := rpcapi.NewHashOrNumber(hashOrNumber)
	if err != nil {
		return false, err
	}
	if !api.mc.CheckState(hn) {
		return false, fmt.Errorf("No meer state at:%s", hashOrNumber)
	}
	return true, nil
}

type PrivateBlockChainAPI struct {
	mc *BlockChain
}

func NewPrivateBlockChainAPI(mc *BlockChain) *PrivateBlockChainAPI {
	return &PrivateBlockChainAPI{mc}
}

func (api *PrivateBlockChainAPI) CalcExportSig(ops string, fee uint64, privKeyHex string) (interface{}, error) {
	sig, err := meerchange.CalcExportSig(meerchange.CalcExportHash(ops, fee), privKeyHex)
	if err != nil {
		return nil, err
	}
	return hex.EncodeToString(sig), nil
}
