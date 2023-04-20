// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package node

import (
	"fmt"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/common/math"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/version"
	"math/big"
	"strconv"
	"time"
)

func (nf *QitmeerFull) apis() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicBlockChainAPI(nf),
			Public:    true,
		},
		{
			NameSpace: cmds.TestNameSpace,
			Service:   NewPrivateBlockChainAPI(nf),
			Public:    false,
		},
		{
			NameSpace: cmds.LogNameSpace,
			Service:   NewPrivateLogAPI(nf),
			Public:    false,
		},
	}
}

type PublicBlockChainAPI struct {
	node *QitmeerFull
}

func NewPublicBlockChainAPI(node *QitmeerFull) *PublicBlockChainAPI {
	return &PublicBlockChainAPI{node}
}

// Return the node info
func (api *PublicBlockChainAPI) GetNodeInfo() (interface{}, error) {
	best := api.node.GetBlockChain().BestSnapshot()
	node := api.node.GetBlockChain().BlockDAG().GetBlock(&best.Hash)
	powNodes := api.node.GetBlockChain().GetCurrentPowDiff(node, pow.MEERXKECCAKV1)
	ret := &json.InfoNodeResult{
		ID:              api.node.GetPeerServer().PeerID().String(),
		Version:         int32(1000000*version.Major + 10000*version.Minor + 100*version.Patch),
		BuildVersion:    version.String(),
		ProtocolVersion: int32(protocol.ProtocolVersion),
		TotalSubsidy:    best.TotalSubsidy,
		TimeOffset:      int64(api.node.GetBlockChain().TimeSource().Offset().Seconds()),
		Connections:     int32(len(api.node.GetPeerServer().Peers().Connected())),
		PowDiff: &json.PowDiff{
			CurrentDiff: getDifficultyRatio(powNodes, api.node.node.Params, pow.MEERXKECCAKV1),
		},
		Network:          params.ActiveNetParams.Name,
		Confirmations:    meerdag.StableConfirmations,
		CoinbaseMaturity: int32(api.node.node.Params.CoinbaseMaturity),
		Modules:          []string{cmds.DefaultServiceNameSpace, cmds.MinerNameSpace, cmds.TestNameSpace, cmds.LogNameSpace, cmds.P2PNameSpace},
	}
	ret.GraphState = marshal.GetGraphStateResult(best.GraphState)
	hostdns := api.node.GetPeerServer().HostDNS()
	if hostdns != nil {
		ret.DNS = hostdns.String()
	}
	if api.node.GetPeerServer().Node() != nil {
		ret.QNR = api.node.GetPeerServer().Node().String()
	}
	if len(api.node.GetPeerServer().HostAddress()) > 0 {
		ret.Addresss = api.node.GetPeerServer().HostAddress()
	}

	// soft forks
	ret.ConsensusDeployment = make(map[string]*json.ConsensusDeploymentDesc)
	ret.ConsensusDeployment["token"] = &json.ConsensusDeploymentDesc{Status: "active"}
	if params.ActiveNetParams.Net == protocol.MainNet {
		cdd := json.ConsensusDeploymentDesc{Status: "inactive", StartHeight: forks.MeerEVMForkMainHeight}
		if forks.IsMeerEVMForkHeight(int64(best.GraphState.GetMainHeight())) {
			cdd.Status = "active"
		}
		ret.ConsensusDeployment["meerevm"] = &cdd
	} else {
		ret.ConsensusDeployment["meerevm"] = &json.ConsensusDeploymentDesc{Status: "active"}
	}

	if api.node.node.Config.Amana {
		ret.ConsensusDeployment["amana"] = &json.ConsensusDeploymentDesc{Status: "active"}
	} else {
		ret.ConsensusDeployment["amana"] = &json.ConsensusDeploymentDesc{Status: "inactive"}
	}

	return ret, nil
}

// getDifficultyRatio returns the proof-of-work difficulty as a multiple of the
// minimum difficulty using the passed bits field from the header of a block.
func getDifficultyRatio(target *big.Int, params *params.Params, powType pow.PowType) float64 {
	instance := pow.GetInstance(powType, 0, []byte{})
	instance.SetParams(params.PowConfig)
	// The minimum difficulty is the max possible proof-of-work limit bits
	// converted back to a number.  Note this is not the same as the proof of
	// work limit directly because the block difficulty is encoded in a block
	// with the compact form which loses precision.
	base := instance.GetSafeDiff(0)
	var difficulty *big.Rat
	if powType == pow.BLAKE2BD || powType == pow.MEERXKECCAKV1 ||
		powType == pow.QITMEERKECCAK256 ||
		powType == pow.X8R16 ||
		powType == pow.X16RV3 ||
		powType == pow.CRYPTONIGHT {
		if target.Cmp(big.NewInt(0)) > 0 {
			difficulty = new(big.Rat).SetFrac(base, target)
		}
	} else {
		difficulty = new(big.Rat).SetFrac(target, base)
	}

	outString := difficulty.FloatString(8)
	diff, err := strconv.ParseFloat(outString, 64)
	if err != nil {
		log.Error(fmt.Sprintf("Cannot get difficulty: %v", err))
		return 0
	}
	return diff
}

// Return the RPC info
func (api *PublicBlockChainAPI) GetRpcInfo() (interface{}, error) {
	rs := api.node.GetRpcServer().ReqStatus
	jrs := []*cmds.JsonRequestStatus{}
	for _, v := range rs {
		jrs = append(jrs, v.ToJson())
	}
	return jrs, nil
}

func (api *PublicBlockChainAPI) GetTimeInfo() (interface{}, error) {
	return fmt.Sprintf("Now:%s offset:%s", roughtime.Now(), roughtime.Offset()), nil
}

func (api *PublicBlockChainAPI) GetNetworkInfo() (interface{}, error) {
	ps := api.node.GetPeerServer()
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
		if p.State.IsConnected() {
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

func (api *PublicBlockChainAPI) GetSubsidy() (interface{}, error) {
	best := api.node.GetBlockChain().BestSnapshot()
	sc := api.node.GetBlockChain().GetSubsidyCache()
	mainHeight := int64(best.GraphState.GetMainHeight())
	binfo := api.node.GetBlockChain().BlockDAG().GetBlueInfo(api.node.GetBlockChain().BlockDAG().GetMainChainTip())

	info := &json.SubsidyInfo{Mode: sc.GetMode(mainHeight), TotalSubsidy: best.TotalSubsidy, BaseSubsidy: params.ActiveNetParams.BaseSubsidy}

	if forks.IsMeerEVMForkHeight(mainHeight) {
		info.TargetTotalSubsidy = forks.MeerEVMForkTotalSubsidy - binfo.GetWeight()
		info.LeftTotalSubsidy = info.TargetTotalSubsidy - int64(info.TotalSubsidy)
		if info.LeftTotalSubsidy < 0 {
			info.TargetTotalSubsidy = 0
		}
	} else if params.ActiveNetParams.TargetTotalSubsidy > 0 {
		info.TargetTotalSubsidy = params.ActiveNetParams.TargetTotalSubsidy
		info.LeftTotalSubsidy = info.TargetTotalSubsidy - int64(info.TotalSubsidy)
		if info.LeftTotalSubsidy < 0 {
			info.TargetTotalSubsidy = 0
		}
		totalTime := time.Duration(info.TargetTotalSubsidy / info.BaseSubsidy * int64(params.ActiveNetParams.TargetTimePerBlock))
		info.TotalTime = totalTime.Truncate(time.Second).String()

		firstMBlock := api.node.GetBlockChain().BlockDAG().GetBlockByOrder(1)
		startTime := time.Unix(api.node.GetBlockChain().BlockDAG().GetBlockData(firstMBlock).GetTimestamp(), 0)
		leftTotalTime := totalTime - time.Since(startTime)
		if leftTotalTime < 0 {
			leftTotalTime = 0
		}
		info.LeftTotalTime = leftTotalTime.Truncate(time.Second).String()
	}
	info.NextSubsidy = sc.CalcBlockSubsidy(binfo)
	return info, nil
}

func (api *PublicBlockChainAPI) GetRpcModules() (interface{}, error) {
	result := []json.KV{
		json.KV{Key: cmds.DefaultServiceNameSpace, Val: false},
		json.KV{Key: cmds.MinerNameSpace, Val: false},
		json.KV{Key: cmds.TestNameSpace, Val: false},
		json.KV{Key: cmds.LogNameSpace, Val: false},
		json.KV{Key: cmds.P2PNameSpace, Val: false},
	}

	for _, m := range api.node.node.Config.Modules {
		for i := 0; i < len(result); i++ {
			if result[i].Key == m {
				result[i].Val = true
			}
		}
	}
	return json.OrderedResult(result), nil
}

func (api *PublicBlockChainAPI) GetMeerDAGInfo() (interface{}, error) {
	mdr := json.MeerDAGInfoResult{}
	md := api.node.GetBlockChain().BlockDAG()
	mdr.Name = md.GetName()
	mdr.Total = md.GetBlockTotal()
	mdr.BlockCacheSize = fmt.Sprintf("%d / %d", md.GetBlockCacheSize(), md.GetMinBlockCacheSize())
	mdr.BlockCacheRate = fmt.Sprintf("%.2f%%", float64(md.GetBlockCacheSize())/float64(mdr.Total)*100)
	mdr.BlockDataCacheSize = fmt.Sprintf("%d / %d", md.GetBlockDataCacheSize(), md.GetMinBlockDataCacheSize())
	return mdr, nil
}

type PrivateBlockChainAPI struct {
	node *QitmeerFull
}

func NewPrivateBlockChainAPI(node *QitmeerFull) *PrivateBlockChainAPI {
	return &PrivateBlockChainAPI{node}
}

// Stop the node
func (api *PrivateBlockChainAPI) Stop() (interface{}, error) {
	select {
	case api.node.GetRpcServer().RequestedProcessShutdown() <- struct{}{}:
	default:
	}
	return "Qitmeer stopping.", nil
}

// Banlist
func (api *PrivateBlockChainAPI) Banlist() (interface{}, error) {
	bl := api.node.GetPeerServer().GetBanlist()
	bls := []*json.GetBanlistResult{}
	for k, v := range bl {
		bls = append(bls, &json.GetBanlistResult{PeerID: k.String(), Bads: v})
	}
	return bls, nil
}

// RemoveBan
func (api *PrivateBlockChainAPI) RemoveBan(id *string) (interface{}, error) {
	ho := ""
	if id != nil {
		ho = *id
	}
	api.node.GetPeerServer().RemoveBan(ho)
	return true, nil
}

// SetRpcMaxClients
func (api *PrivateBlockChainAPI) SetRpcMaxClients(max int) (interface{}, error) {
	if max <= 0 {
		err := fmt.Errorf("error:Must greater than 0 (cur max =%d)", api.node.node.Config.RPCMaxClients)
		return api.node.node.Config.RPCMaxClients, err
	}
	api.node.node.Config.RPCMaxClients = max
	return api.node.node.Config.RPCMaxClients, nil
}

type PrivateLogAPI struct {
	node *QitmeerFull
}

func NewPrivateLogAPI(node *QitmeerFull) *PrivateLogAPI {
	return &PrivateLogAPI{node}
}

// set log
func (api *PrivateLogAPI) SetLogLevel(level string) (interface{}, error) {
	err := common.ParseAndSetDebugLevels(level)
	if err != nil {
		return nil, err
	}
	api.node.GetVMService().SetLogLevel(level)
	return level, nil
}
