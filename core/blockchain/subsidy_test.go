package blockchain

import (
	"fmt"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"testing"
	"time"
)

// TestEstimateSupply ensures the supply estimation function defined by MeerEVM-fork works as expected.
func TestEstimateSupplyByMeerEVMFork(t *testing.T) {
	params.ActiveNetParams = &params.MainNetParam
	param := params.MainNetParam.Params
	maxTestInterval := int64(28)
	endBlockHeight := int64(param.MeerEVMForkBlock.Int64() + forks.SubsidyReductionInterval*maxTestInterval + 1)
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

	baseSubsidy := int64(1000000000)
	forkSubsidy := int64(param.MeerEVMForkBlock.Int64() * baseSubsidy)

	type testData struct {
		height             int64
		expectSubsidy      int64
		expectMode         string
		expectTotalSubsidy int64
	}
	tests := []testData{
		{height: 0, expectSubsidy: 0, expectTotalSubsidy: 0, expectMode: "static"},
		{height: 1, expectSubsidy: blockOneSubsidy, expectTotalSubsidy: 1000000000, expectMode: "static"},
		{height: 2, expectSubsidy: blockTwoSubsidy, expectTotalSubsidy: 2000000000, expectMode: "static"},
	}

	firstIndex := len(tests)
	for i := int64(0); i < maxTestInterval; i++ {
		td := testData{height: param.MeerEVMForkBlock.Int64() + forks.SubsidyReductionInterval*i, expectSubsidy: subsidyCache.subsidyCache[uint64(i)], expectMode: "meerevmfork"}
		if i == 0 {
			td.expectTotalSubsidy = forkSubsidy
			td.expectSubsidy = baseSubsidy
		} else {
			pidx := firstIndex + int(i) - 1
			td.expectTotalSubsidy = tests[pidx].expectTotalSubsidy + tests[pidx].expectSubsidy*(forks.SubsidyReductionInterval-1) + td.expectSubsidy
		}
		tests = append(tests, td)
	}

	getTime := func(height int64) string {
		bt := param.TargetTimePerBlock * time.Duration(height)
		oneYear := time.Hour * 24 * 365
		if bt < oneYear {
			return bt.String()
		}
		return fmt.Sprintf("%d years", bt/oneYear)
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
		t.Logf("height:%d subsidy:%d total:%d left:%d mode:%s time:%s", test.height, test.expectSubsidy,
			test.expectTotalSubsidy, int64(forks.MeerEVMForkTotalSubsidy)-test.expectTotalSubsidy, test.expectMode, getTime(test.height))
	}
}
