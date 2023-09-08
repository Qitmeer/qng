package common

import (
	"github.com/ethereum/go-ethereum/metrics"
)

const (
	// ingressMeterName is the prefix of the per-packet inbound metrics.
	ingressMeterName = "p2p/ingress/qng"

	// egressMeterName is the prefix of the per-packet outbound metrics.
	egressMeterName = "p2p/egress/qng"

)

var (
	IngressConnectMeter = metrics.NewRegisteredMeter("p2p/serves/qng", nil)
	IngressTrafficMeter = metrics.NewRegisteredMeter(ingressMeterName, nil)
	EgressConnectMeter  = metrics.NewRegisteredMeter("p2p/dials/qng", nil)
	EgressTrafficMeter  = metrics.NewRegisteredMeter(egressMeterName, nil)
	ActivePeerGauge     = metrics.NewRegisteredGauge("p2p/peers/qng", nil)
)