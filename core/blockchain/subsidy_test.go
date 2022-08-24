package blockchain

import (
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"testing"
)

// TestEstimateSupply ensures the supply estimation function defined by MeerEVM-fork works as expected.
func TestEstimateSupplyByMeerEVMFork(t *testing.T) {
	params.ActiveNetParams = &params.MainNetParam
	param := params.MainNetParam.Params
	endBlockHeight := int64(62621744)
	bis := map[int64]*meerdag.BlueInfo{}

	subsidyCache := NewSubsidyCache(0, param)
	calcBlockSubsidy := func(height int64) int64 {
		if height == 0 {
			return 0
		}
		bi, ok := bis[height]
		if !ok {
			t.Fatalf("No bi:%d", height)
		}
		return subsidyCache.CalcBlockSubsidy(bi)
	}

	for i := int64(0); i <= endBlockHeight; i++ {
		var weight int64
		var bs int64
		if i == 0 {
			weight = 0
		} else if i == 1 {
			weight = 0
		} else {
			pheight := i - 1
			pbi, ok := bis[pheight]
			if !ok {
				t.Fatal("No test bi")
			}
			bs = calcBlockSubsidy(pheight)
			weight = pbi.GetWeight() + bs
		}
		bis[i] = meerdag.NewBlueInfo(0, 0, weight, i)
	}

	blockOneSubsidy := calcBlockSubsidy(1)
	blockTwoSubsidy := calcBlockSubsidy(2)

	tests := []struct {
		height             int64
		expectSubsidy      int64
		expectMode         string
		expectTotalSubsidy int64
	}{
		{height: 0, expectSubsidy: 0, expectTotalSubsidy: 0, expectMode: "static"},
		{height: 1, expectSubsidy: blockOneSubsidy, expectTotalSubsidy: 1000000000, expectMode: "static"},
		{height: 2, expectSubsidy: blockTwoSubsidy, expectTotalSubsidy: 2000000000, expectMode: "static"},
		{height: forks.MeerEVMForkMainHeight, expectSubsidy: 1000000000, expectTotalSubsidy: 959000000000000, expectMode: "meerevmfork"},
		{height: 4686074, expectSubsidy: 772047951, expectTotalSubsidy: 4244275010176525, expectMode: "meerevmfork"},  // Half of the total subsidy
		{height: 10635800, expectSubsidy: 498314832, expectTotalSubsidy: 7963647733148752, expectMode: "meerevmfork"}, // 10 years
		{height: 42154520, expectSubsidy: 51550180, expectTotalSubsidy: 14201480957219300, expectMode: "meerevmfork"}, // 40 years
		{height: 62621742, expectSubsidy: 11821302, expectTotalSubsidy: 14756274989081626, expectMode: "meerevmfork"}, // before end point
		{height: 62621743, expectSubsidy: 10918374, expectTotalSubsidy: 14756275000000000, expectMode: "meerevmfork"}, // end point
	}

	for _, test := range tests {
		if test.expectSubsidy >= 0 {
			gotSupply := calcBlockSubsidy(test.height)
			if gotSupply != test.expectSubsidy {
				t.Fatalf("calcBlockSubsidy (height %d): did not get "+
					"expected supply - got %d, want %d", test.height,
					gotSupply, test.expectSubsidy)
			}
		}
		if len(test.expectMode) > 0 {
			cm := subsidyCache.GetMode(test.height)
			if cm != test.expectMode {
				t.Fatalf("subsidy mode is %s ,want %s", cm, test.expectMode)
			}
		}
		if test.expectTotalSubsidy > 0 {
			nbi, ok := bis[test.height+1]
			if !ok {
				t.Fatal("No test bi")
			}
			if nbi.GetWeight() != test.expectTotalSubsidy {
				t.Fatalf("height: %d,total subsidy:%d != %d", test.height, nbi.GetWeight(), test.expectTotalSubsidy)
			}
		}
	}
}
