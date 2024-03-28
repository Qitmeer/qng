// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package node

import (
	js "encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/blockchain"
	"math/big"
	"strconv"
	"time"

	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/common"
	"github.com/Qitmeer/qng/version"
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
		Connections:     int32(len(api.node.GetPeerServer().Peers().Active())),
		PowDiff: &json.PowDiff{
			CurrentDiff: getDifficultyRatio(powNodes, api.node.node.Params, pow.MEERXKECCAKV1),
		},
		Network:          params.ActiveNetParams.Name,
		Confirmations:    meerdag.StableConfirmations,
		CoinbaseMaturity: int32(api.node.node.Params.CoinbaseMaturity),
		Modules:          []string{cmds.DefaultServiceNameSpace, cmds.MinerNameSpace, cmds.TestNameSpace, cmds.LogNameSpace, cmds.P2PNameSpace, cmds.WalletNameSpace},
	}
	ret.GraphState = marshal.GetGraphStateResult(best.GraphState)
	ret.StateRoot = best.StateRoot.String()
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
		startTime := time.Unix(api.node.GetBlockChain().GetBlockNode(firstMBlock).GetTimestamp(), 0)
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
		json.KV{Key: cmds.WalletNameSpace, Val: false},
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
	mdr.BlockCacheSize = md.GetBlockCacheSize()
	mdr.BlockCacheHeightSize = md.GetMinBlockCacheSize()
	mdr.BlockCacheRate = fmt.Sprintf("%.2f%%", float64(md.GetBlockCacheSize())/float64(mdr.Total)*100)
	mdr.BlockDataCacheSize = fmt.Sprintf("%d / %d", md.GetBlockDataCacheSize(), md.GetMinBlockDataCacheSize())
	mdr.AnticoneSize = md.GetInstance().(*meerdag.Phantom).AnticoneSize()
	return mdr, nil
}

func (api *PublicBlockChainAPI) GetDatabaseInfo() (interface{}, error) {
	info, _ := api.node.db.GetInfo()
	ret := json.OrderedResult{
		json.KV{Key: "name", Val: api.node.db.Name()},
		json.KV{Key: "info", Val: info.String()},
		json.KV{Key: "engine", Val: api.node.db.DBEngine()},
		json.KV{Key: "snapshot", Val: api.node.db.SnapshotInfo()},
	}
	return ret, nil
}

func (api *PublicBlockChainAPI) GetChainInfo(lastCount int, start *int64, end *int64) (interface{}, error) {
	if !api.node.GetBlockChain().IsNearlySynced() {
		return nil, fmt.Errorf("Busy, try again later")
	}
	count := meerdag.MinBlockPruneSize
	if lastCount > 1 {
		count = lastCount
	}
	md := api.node.GetBlockChain().BlockDAG()
	var startBlock meerdag.IBlock
	var endBlock meerdag.IBlock
	var info *json.ChainInfoResult
	mainTip := md.GetMainChainTip()
	totalTxs := 0
	emptyBlocks := 0
	startOrder := uint(0)
	if start != nil {
		startOrder = uint(*start)
	}
	endOrder := uint(0)
	if end != nil {
		endOrder = uint(*end)
	}
	if startOrder > 0 {
		if endOrder > 0 {
			endBlock = md.GetBlockByOrder(endOrder)
			if endBlock == nil {
				return nil, fmt.Errorf("No end block by order:%d", endOrder)
			}
		} else {
			endOrder = mainTip.GetOrder()
		}
		if startOrder >= endOrder {
			return nil, fmt.Errorf("Invalid start block by order:%d", startOrder)
		}
		startBlock = md.GetBlockByOrder(startOrder)
		if startBlock == nil {
			return nil, fmt.Errorf("No start block by order:%d", startOrder)
		}

		info = &json.ChainInfoResult{Count: 0}
		var block meerdag.IBlock
		for i := startOrder + 1; i <= endOrder; i++ {
			block = md.GetBlockByOrder(i)
			if block == nil {
				return nil, fmt.Errorf("No block by order:%d", i)
			}
			blockNode := api.node.GetBlockChain().GetBlockNode(block)
			if blockNode == nil {
				return true, nil
			}
			info.Count++
			totalTxs += blockNode.GetPriority()
			if blockNode.GetPriority() <= 1 {
				emptyBlocks++
			}
			if endBlock != nil {
				continue
			}
			if info.Count >= uint64(count) {
				endBlock = block
				break
			}
		}
		if endBlock == nil {
			endBlock = block
		}
	} else {
		endBlock = mainTip
		info = &json.ChainInfoResult{Count: 0}
		var blockNode *blockchain.BlockNode
		err := md.Foreach(endBlock, uint(count), meerdag.All, func(block meerdag.IBlock) (bool, error) {
			if block.GetID() <= 0 {
				return true, nil
			}
			blockNode = api.node.GetBlockChain().GetBlockNode(block)
			if blockNode == nil {
				return true, nil
			}
			info.Count++
			totalTxs += blockNode.GetPriority()
			if blockNode.GetPriority() <= 1 {
				emptyBlocks++
			}
			startBlock = block
			return true, nil
		})
		if err != nil {
			return nil, err
		}
		if blockNode != nil {
			totalTxs -= blockNode.GetPriority()
			if totalTxs < 0 {
				totalTxs = 0
			}
			if blockNode.GetPriority() <= 1 {
				emptyBlocks--
			}
		}
		endNode := api.node.GetBlockChain().GetBlockNode(endBlock)
		if endNode == nil {
			return nil, fmt.Errorf("No block:%s", endBlock.GetHash().String())
		}
		totalTxs += endNode.GetPriority()
		if endNode.GetPriority() <= 1 {
			emptyBlocks++
		}
	}

	if info.Count <= 0 {
		return nil, fmt.Errorf("No blocks")
	}

	info.Start = fmt.Sprintf("%s (order:%d)", startBlock.GetHash().String(), startBlock.GetOrder())
	info.End = fmt.Sprintf("%s (order:%d)", endBlock.GetHash().String(), endBlock.GetOrder())
	startNode := api.node.GetBlockChain().GetBlockHeader(startBlock)
	if startNode == nil {
		return nil, fmt.Errorf("No block:%s", startBlock.GetHash().String())
	}
	endNode := api.node.GetBlockChain().GetBlockNode(endBlock)
	if endNode == nil {
		return nil, fmt.Errorf("No block:%s", endBlock.GetHash().String())
	}

	totalTime := endNode.GetTimestamp() - startNode.Timestamp.Unix()
	if totalTime < 0 {
		totalTime = 0
	}
	if totalTime <= 0 {
		return nil, fmt.Errorf("Time is too short")
	}
	info.BlocksPerSecond = float64(info.Count) / float64(totalTime)
	info.TxsPerSecond = float64(totalTxs) / float64(totalTime)

	totalHeight := int64(endBlock.GetHeight()) - int64(startBlock.GetHeight())
	if totalHeight < 0 {
		totalHeight = 0
	}
	if totalHeight > 0 {
		secondPerHeight := time.Duration(totalTime) * time.Second / time.Duration(totalHeight)
		info.SecondPerHeight = secondPerHeight.String()
		info.Concurrency = float64(info.Count) / float64(totalHeight)
	}
	info.EmptyBlockRate = fmt.Sprintf("%d%%", uint64(emptyBlocks*100)/info.Count)
	info.ProcessQueueSize = int32(api.node.GetBlockChain().ProcessQueueSize())
	return *info, nil
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

// SetRpcMaxClients
func (api *PrivateBlockChainAPI) SetRpcMaxClients(max int) (interface{}, error) {
	if max <= 0 {
		err := fmt.Errorf("error:Must greater than 0 (cur max =%d)", api.node.node.Config.RPCMaxClients)
		return api.node.node.Config.RPCMaxClients, err
	}
	api.node.node.Config.RPCMaxClients = max
	return api.node.node.Config.RPCMaxClients, nil
}

func (api *PrivateBlockChainAPI) GetConfig() (interface{}, error) {
	cs, err := js.Marshal(*api.node.node.Config)
	if err != nil {
		return nil, err
	}
	return string(cs), nil
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
	eth.InitLog(level, api.node.node.Config.DebugPrintOrigins)
	return level, nil
}
