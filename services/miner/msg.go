package miner

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/services/mining"
)

type StartCPUMiningMsg struct {
}

type CPUMiningGenerateMsg struct {
	discreteNum int
	block       chan *hash.Hash
	powType     pow.PowType
}

type BlockChainChangeMsg struct {
}

type MempoolChangeMsg struct {
}

type gbtResponse struct {
	result interface{}
	err    error
}

type GBTMiningMsg struct {
	request *json.TemplateRequest
	reply   chan *gbtResponse
}

type RemoteMiningMsg struct {
	powType       pow.PowType
	coinbaseFlags mining.CoinbaseFlags
	reply         chan *gbtResponse
}
