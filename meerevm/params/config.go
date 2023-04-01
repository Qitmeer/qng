package params

import (
	"github.com/Qitmeer/qng/core/protocol"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	eparams "github.com/ethereum/go-ethereum/params"
	"math/big"
)

var (
	// QNG
	QngMainnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.QngMainnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         nil,
		Ethash:              new(eparams.EthashConfig),
	}

	QngTestnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.QngTestnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         nil,
		Ethash:              new(eparams.EthashConfig),
	}

	QngMixnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.QngMixnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         nil,
		Ethash:              new(eparams.EthashConfig),
	}

	QngPrivnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.QngPrivnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		Ethash:              new(eparams.EthashConfig),
	}

	// Amana
	AmanaChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.AmanaChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		Clique: &eparams.CliqueConfig{
			Period: 3,
			Epoch:  100,
		},
	}

	AmanaTestnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.AmanaTestnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		Clique: &eparams.CliqueConfig{
			Period: 3,
			Epoch:  100,
		},
	}

	AmanaMixnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.AmanaMixnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		Clique: &eparams.CliqueConfig{
			Period: 3,
			Epoch:  100,
		},
	}

	AmanaPrivnetChainConfig = &eparams.ChainConfig{
		ChainID:             eparams.AmanaPrivnetChainConfig.ChainID,
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArrowGlacierBlock:   big.NewInt(0),
		GrayGlacierBlock:    big.NewInt(0),
		Clique: &eparams.CliqueConfig{
			Period: 3,
			Epoch:  100,
		},
	}
)

func ChainConfig(ct ChainType) *eparams.ChainConfig {
	switch qparams.ActiveNetParams.Net {
	case protocol.MainNet:
		switch ct {
		case Qng:
			return QngMainnetChainConfig
		case Amana:
			return AmanaChainConfig
		}
	case protocol.TestNet:
		switch ct {
		case Qng:
			return QngTestnetChainConfig
		case Amana:
			return AmanaTestnetChainConfig
		}
	case protocol.MixNet:
		switch ct {
		case Qng:
			return QngMixnetChainConfig
		case Amana:
			return AmanaMixnetChainConfig
		}
	case protocol.PrivNet:
		switch ct {
		case Qng:
			return QngPrivnetChainConfig
		case Amana:
			return AmanaPrivnetChainConfig
		}
	}

	return nil
}
