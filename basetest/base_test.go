package basetest

import (
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/params"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvmGenesis(t *testing.T) {
	assert.Equal(t, meer.MainAllocHash, meer.BuildGenesisHash(params.MainNetParams.Name),
		params.MainNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, meer.MixAllocHash, meer.BuildGenesisHash(params.MixNetParams.Name),
		params.MixNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, meer.TestAllocHash, meer.BuildGenesisHash(params.TestNetParams.Name),
		params.TestNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, meer.PrivAllocHash, meer.BuildGenesisHash(params.PrivNetParams.Name),
		params.PrivNetParam.Name+" genesis hash not equal latest")
}
