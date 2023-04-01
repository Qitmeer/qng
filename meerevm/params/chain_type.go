package params

import (
	"fmt"
	eparams "github.com/ethereum/go-ethereum/params"
)

type ChainType uint32

const (
	Qng ChainType = iota
	Amana
	Flana
	Mizana
)

func (ct ChainType) String() string {
	switch ct {
	case Qng:
		return eparams.NetworkNames[eparams.QngMainnetChainConfig.ChainID.String()]
	case Amana:
		return eparams.NetworkNames[eparams.AmanaChainConfig.ChainID.String()]
	case Flana:
		return eparams.NetworkNames[eparams.FlanaChainConfig.ChainID.String()]
	case Mizana:
		return eparams.NetworkNames[eparams.MizanaChainConfig.ChainID.String()]
	}
	return fmt.Sprintf("Unknown ChainType (%d)", uint32(ct))
}
