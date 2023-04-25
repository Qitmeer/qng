// Copyright (c) 2017-2018 The qitmeer developers

package miner

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/rpc"
	"github.com/Qitmeer/qng/rpc/api"
	"github.com/Qitmeer/qng/rpc/client/cmds"
	"github.com/Qitmeer/qng/services/mining"
	"time"
)

const (
	SubmitInterval = time.Second
)

func (m *Miner) APIs() []api.API {
	return []api.API{
		{
			NameSpace: cmds.DefaultServiceNameSpace,
			Service:   NewPublicMinerAPI(m),
			Public:    true,
		},
		{
			NameSpace: cmds.MinerNameSpace,
			Service:   NewPrivateMinerAPI(m),
			Public:    false,
		},
	}
}

type MiningStats struct {
	LastGBTTime          time.Time `json:"last_gbt_time"`
	LastSubmit           time.Time `json:"last_submit_time"`
	Last100GbtTimes      []int64   `json:"last_100_gbt_times"`
	Last100GbtPerTime    float64   `json:"last_100_gbt_per_time"`
	Last100SubmitTimes   []int64   `json:"last_100_submit_times"`
	Last100SubmitPerTime float64   `json:"last_100_submit_per_time"`
	SubmitPerTime        float64   `json:"submit_per_time"`
	GbtPerTime           float64   `json:"gbt_per_time"`
}

type PublicMinerAPI struct {
	miner *Miner
	stats MiningStats
}

func NewPublicMinerAPI(m *Miner) *PublicMinerAPI {
	pmAPI := &PublicMinerAPI{miner: m, stats: MiningStats{
		LastSubmit:         time.Now(),
		LastGBTTime:        time.Now(),
		Last100GbtTimes:    make([]int64, 0),
		Last100SubmitTimes: make([]int64, 0),
	}}
	return pmAPI
}
func (api *PublicMinerAPI) StatsGbt(currentReqMillSec int64) {
	if len(api.stats.Last100GbtTimes) >= 100 {
		api.stats.Last100GbtTimes = api.stats.Last100GbtTimes[len(api.stats.Last100GbtTimes)-99:]
	}
	api.stats.Last100GbtTimes = append(api.stats.Last100GbtTimes, currentReqMillSec)
	sum := int64(0)
	for _, v := range api.stats.Last100GbtTimes {
		sum += v
	}
	api.stats.Last100GbtPerTime = float64(sum) / float64(len(api.stats.Last100GbtTimes)) / 1000
	api.stats.GbtPerTime = (api.stats.GbtPerTime + float64(currentReqMillSec)) / 2 / 1000
}
func (api *PublicMinerAPI) StatsSubmit(currentReqMillSec int64) {
	if len(api.stats.Last100SubmitTimes) >= 100 {
		api.stats.Last100SubmitTimes = api.stats.Last100SubmitTimes[len(api.stats.Last100SubmitTimes)-99:]
	}
	api.stats.Last100SubmitTimes = append(api.stats.Last100SubmitTimes, currentReqMillSec)
	sum := int64(0)
	for _, v := range api.stats.Last100SubmitTimes {
		sum += v
	}
	api.stats.Last100SubmitPerTime = float64(sum) / float64(len(api.stats.Last100SubmitTimes)) / 1000
	api.stats.SubmitPerTime = (api.stats.SubmitPerTime + float64(currentReqMillSec)) / 2 / 1000
}

// func (api *PublicMinerAPI) GetBlockTemplate(request *mining.TemplateRequest) (interface{}, error){
func (api *PublicMinerAPI) GetBlockTemplate(capabilities []string, powType byte) (interface{}, error) {
	// Set the default mode and override it if supplied.
	mode := "template"
	request := json.TemplateRequest{Mode: mode, Capabilities: capabilities, PowType: powType}
	if err := api.checkGBTTime(); err != nil {
		return nil, err
	}
	switch mode {
	case "template":
		start := time.Now().UnixMilli()
		log.Debug("gbtstart")
		data, err := handleGetBlockTemplateRequest(api, &request)
		api.StatsGbt(time.Now().UnixMilli() - start)
		log.Debug("gbtend")
		return data, err
	case "proposal":
		//TODO LL, will be added
		//return handleGetBlockTemplateProposal(s, request)
	}
	return nil, rpc.RpcInvalidError("Invalid mode")
}

// GetMiningStats func (api *PublicMinerAPI) GetMiningStats() (interface{}, error){
func (api *PublicMinerAPI) GetMiningStats() (interface{}, error) {
	return api.stats, nil
}

// LL
// handleGetBlockTemplateRequest is a helper for handleGetBlockTemplate which
// deals with generating and returning block templates to the caller. In addition,
// it detects the capabilities reported by the caller
// in regards to whether or not it supports creating its own coinbase (the
// coinbasetxn and coinbasevalue capabilities) and modifies the returned block
// template accordingly.
func handleGetBlockTemplateRequest(api *PublicMinerAPI, request *json.TemplateRequest) (interface{}, error) {
	reply := make(chan *gbtResponse)
	err := api.miner.GBTMining(request, reply)
	if err != nil {
		return nil, err
	}
	resp := <-reply
	return resp.result, resp.err
}

// LL
// Attempts to submit new block to network.
// See https://en.bitcoin.it/wiki/BIP_0022 for full specification
func (api *PublicMinerAPI) SubmitBlock(hexBlock string) (interface{}, error) {
	if err := api.checkSubmitLimit(); err != nil {
		return nil, err
	}
	api.stats.LastSubmit = time.Now()
	// Deserialize the hexBlock.
	m := api.miner

	if len(hexBlock)%2 != 0 {
		hexBlock = "0" + hexBlock
	}
	serializedBlock, err := hex.DecodeString(hexBlock)

	if err != nil {
		return nil, rpc.RpcDecodeHexError(hexBlock)
	}
	block, err := types.NewBlockFromBytes(serializedBlock)
	if err != nil {
		return nil, rpc.RpcDeserializationError("Block decode failed: %s", err.Error())
	}

	// Because it's asynchronous, so you must ensure that all tips are referenced
	if len(block.Block().Transactions) <= 0 {
		return nil, fmt.Errorf("block is illegal")
	}

	height, err := blockchain.ExtractCoinbaseHeight(block.Block().Transactions[0])
	if err != nil {
		return nil, err
	}

	block.SetHeight(uint(height))
	start := time.Now().UnixMilli()
	log.Debug("submitstart", "blockhash", block.Block().BlockHash(), "txcount", len(block.Block().Transactions))
	res, err := m.submitBlock(block)
	api.StatsSubmit(time.Now().UnixMilli() - start)
	log.Debug("submitend", "blockhash", block.Block().BlockHash(), "txcount",
		len(block.Block().Transactions), "res", res, "err", err)
	return res, err
}

func (api *PublicMinerAPI) GetMinerInfo() (interface{}, error) {
	if !api.miner.IsEnable() {
		return nil, fmt.Errorf("Miner is disable. You can enable by --miner.")
	}
	if api.miner.template == nil || api.miner.worker == nil {
		return nil, fmt.Errorf("Not ready")
	}
	result := json.MinerInfoResult{}
	result.Timestamp = api.miner.template.Block.Header.Timestamp.String()
	result.Height = api.miner.template.Height
	result.Pow = pow.GetPowName(api.miner.powType)
	result.Difficulty = fmt.Sprintf("%x", api.miner.template.Block.Header.Difficulty)
	result.Target = fmt.Sprintf("%064x", pow.CompactToBig(api.miner.template.Block.Header.Difficulty))
	result.Coinbase = api.miner.coinbaseAddress.String()
	result.CoinbaseFlags = string(api.miner.coinbaseFlags)
	result.TotalSubmit = api.miner.totalSubmit
	result.SuccessSubmit = api.miner.successSubmit
	if api.miner.worker != nil {
		result.Running = api.miner.worker.IsRunning()
		result.Type = api.miner.worker.GetType()
	}

	return &result, nil
}

func (api *PublicMinerAPI) GetRemoteGBT(powType byte, extraNonce *bool) (interface{}, error) {
	reply := make(chan *gbtResponse)
	coinbaseFlags := mining.CoinbaseFlagsStatic
	if extraNonce != nil && *extraNonce {
		coinbaseFlags = mining.CoinbaseFlagsDynamic
	}
	err := api.miner.RemoteMining(pow.PowType(powType), coinbaseFlags, reply)
	if err != nil {
		return nil, err
	}
	resp := <-reply
	return resp.result, resp.err
}

func (api *PublicMinerAPI) SubmitBlockHeader(hexBlockHeader string, extraNonce *uint64) (interface{}, error) {
	if err := api.checkSubmitLimit(); err != nil {
		return nil, err
	}
	api.stats.LastSubmit = time.Now()
	// Deserialize the hexBlock.
	m := api.miner

	if len(hexBlockHeader)%2 != 0 {
		hexBlockHeader = "0" + hexBlockHeader
	}
	serializedBlockHeader, err := hex.DecodeString(hexBlockHeader)
	if err != nil {
		return nil, rpc.RpcDecodeHexError(hexBlockHeader)
	}
	var header types.BlockHeader
	err = header.Deserialize(bytes.NewReader(serializedBlockHeader))
	if err != nil {
		return nil, err
	}
	md := uint64(0)
	if extraNonce != nil {
		md = *extraNonce
	}
	return m.submitBlockHeader(&header, md)
}

func (api *PublicMinerAPI) checkSubmitLimit() error {
	if time.Since(api.stats.LastSubmit) < SubmitInterval {
		return fmt.Errorf("Submission interval Limited:%s < %s\n", time.Since(api.stats.LastSubmit), SubmitInterval)
	}
	return nil
}

func (api *PublicMinerAPI) checkGBTTime() error {
	if time.Since(api.stats.LastGBTTime) < params.ActiveNetParams.TargetTimePerBlock {
		log.Trace("Client in sunc, qitmeer is sync tx...")
		return rpc.RPCClientInInitialDownloadError("Client in initial download ",
			"qitmeer is downloading tx...")
	}
	return nil
}

// PrivateMinerAPI provides private RPC methods to control the miner.
type PrivateMinerAPI struct {
	miner *Miner
}

func NewPrivateMinerAPI(m *Miner) *PrivateMinerAPI {
	pmAPI := &PrivateMinerAPI{miner: m}
	return pmAPI
}

func (api *PrivateMinerAPI) Generate(numBlocks uint32, powType pow.PowType) ([]string, error) {
	// Respond with an error if the client is requesting 0 blocks to be generated.
	if numBlocks == 0 {
		return nil, rpc.RpcInternalError("Invalid number of blocks",
			"Configuration")
	}
	if numBlocks > 3000 {
		return nil, fmt.Errorf("error, more than 1000")
	}

	// Create a reply
	reply := []string{}

	blockHashC := make(chan *hash.Hash)
	err := api.miner.CPUMiningGenerate(int(numBlocks), blockHashC, powType)
	if err != nil {
		return nil, err
	}
	for i := uint32(0); i < numBlocks; i++ {
		select {
		case blockHash := <-blockHashC:
			if blockHash == nil {
				break
			}
			// Mine the correct number of blocks, assigning the hex representation of the
			// hash of each one to its place in the reply.
			reply = append(reply, blockHash.String())
		}
	}
	if len(reply) <= 0 {
		return nil, fmt.Errorf("No blocks")
	}
	return reply, nil
}
