package basetest

import (
	"github.com/Qitmeer/qng/meerevm/chain"
	"github.com/Qitmeer/qng/params"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvmGenesis(t *testing.T) {
	assert.Equal(t, chain.MainAllocHash, chain.BuildGenesisHash(params.MainNetParams.Name),
		params.MainNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, chain.MixAllocHash, chain.BuildGenesisHash(params.MixNetParams.Name),
		params.MixNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, chain.TestAllocHash, chain.BuildGenesisHash(params.TestNetParams.Name),
		params.TestNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, chain.PrivAllocHash, chain.BuildGenesisHash(params.PrivNetParams.Name),
		params.PrivNetParam.Name+" genesis hash not equal latest")
}
