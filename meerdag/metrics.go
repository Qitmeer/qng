package meerdag

import (
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	mainOrderGauge  = metrics.NewRegisteredGauge("meerdag/mainorder", nil)
	mainHeightGauge = metrics.NewRegisteredGauge("meerdag/mainheight", nil)
	mainLayerGauge  = metrics.NewRegisteredGauge("meerdag/mainlayer", nil)

	tipsTotalGauge   = metrics.NewRegisteredGauge("meerdag/tips/total", nil)
	unsequencedGauge = metrics.NewRegisteredGauge("meerdag/unsequenced", nil)
	reorganizeGauge  = metrics.NewRegisteredGauge("meerdag/reorganize", nil)
)
