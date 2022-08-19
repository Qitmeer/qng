package basetest

import (
	"github.com/Qitmeer/qng/meerevm/chain"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvmGenesis(t *testing.T) {
	assert.Equal(t, chain.Check(), true, "genesis hash not equal latest")
}
