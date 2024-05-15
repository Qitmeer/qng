package meer

import (
	"testing"

	"github.com/Qitmeer/qng/params"
	"github.com/stretchr/testify/assert"
)

func TestGenesisHash(t *testing.T) {
	assert.Equal(t, MainNetGenesisHash, Genesis(params.MainNetParams.Net).ToBlock().Hash().String(),
		params.MainNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, MixNetGenesisHash, Genesis(params.MixNetParams.Net).ToBlock().Hash().String(),
		params.MixNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, TestNetGenesisHash, Genesis(params.TestNetParams.Net).ToBlock().Hash().String(),
		params.TestNetParams.Name+" genesis hash not equal latest")
	assert.Equal(t, PrivNetGenesisHash, Genesis(params.PrivNetParams.Net).ToBlock().Hash().String(),
		params.PrivNetParam.Name+" genesis hash not equal latest")
}
