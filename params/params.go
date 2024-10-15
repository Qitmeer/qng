// Copyright (c) 2017-2018 The qitmeer developers
// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package params

import (
	"encoding/hex"
	"errors"
	eparams "github.com/ethereum/go-ethereum/params"
	"math/big"
	"strings"
	"time"

	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/ledger"
)

// CheckForDuplicateHashes checks for duplicate hashes when validating blocks.
// Because of the rule inserting the height into the second (nonce) txOut, there
// should never be a duplicate transaction hash that overwrites another. However,
// because there is a 2^128 chance of a collision, the paranoid user may wish to
// turn this feature on.
var CheckForDuplicateHashes = false

// SigHashOptimization is an optimization for verification of transactions that
// do CHECKSIG operations with hashType SIGHASH_ALL. Although there should be no
// consequences to daemons that are simply running a node, it may be the case
// that you could cause database corruption if you turn this code on, create and
// manipulate your own MsgTx, then include them in blocks. For safety, if you're
// using the daemon with wallet or mining with the daemon this should be disabled.
// If you believe that any MsgTxs in your daemon will be used mutably, do NOT
// turn on this feature. It is disabled by default.
// This feature is considered EXPERIMENTAL, enable at your own risk!
var SigHashOptimization = false

// CPUMinerThreads is the default number of threads to utilize with the
// CPUMiner when mining.
var CPUMinerThreads = 1

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
//
// Each checkpoint is selected based upon several factors.  See the
// documentation for blockchain.IsCheckpointCandidate for details on the
// selection criteria.
type Checkpoint struct {
	Layer uint64
	Hash  *hash.Hash
}

// Params defines a qitmeer network by its parameters.  These parameters may be
// used by qitmeer applications to differentiate networks as well as addresses
// and keys for one network from those intended for use on another network.
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Net defines the magic bytes used to identify the network.
	Net protocol.Network

	// TCPPort defines the default peer-to-peer tcp port for the network.
	DefaultPort string

	// DefaultUDPPort defines the default peer-to-peer udp port for the network.
	DefaultUDPPort int

	// Bootstrap defines a list of boot node for the network that are used
	// as one method to discover peers.
	Bootstrap []string

	// GenesisBlock defines the first block of the chain.
	GenesisBlock *types.SerializedBlock

	// GenesisHash is the starting block hash.
	GenesisHash *hash.Hash

	// PowConfig defines the highest allowed proof of work value for a block or lowest difficulty for a block
	PowConfig *pow.PowConfig

	// WorkDiffAlpha is the stake difficulty EMA calculation alpha (smoothing)
	// value. It is different from a normal EMA alpha. Closer to 1 --> smoother.
	WorkDiffAlpha int64

	// WorkDiffWindowSize is the number of windows (intervals) used for calculation
	// of the exponentially weighted average.
	WorkDiffWindowSize int64

	// WorkDiffWindows is the number of windows (intervals) used for calculation
	// of the exponentially weighted average.
	WorkDiffWindows int64

	// CoinbaseMaturity is the number of blocks required before newly mined
	// coins (coinbase transactions) can be spent.
	CoinbaseMaturity uint16

	// TargetTimespan is the desired amount of time that should elapse
	// before the block difficulty requirement is examined to determine how
	// it should be changed in order to maintain the desired block
	// generation rate.
	TargetTimespan time.Duration

	// TargetTimePerBlock is the desired amount of time to generate each
	// block.
	TargetTimePerBlock time.Duration

	// RetargetAdjustmentFactor is the adjustment factor used to limit
	// the minimum and maximum amount of adjustment that can occur between
	// difficulty retargets.
	RetargetAdjustmentFactor int64

	// ReduceMinDifficulty defines whether the network should reduce the
	// minimum required difficulty after a long enough period of time has
	// passed without finding a block.  This is really only useful for test
	// networks and should not be set on a main network.
	ReduceMinDifficulty bool

	// MinDiffReductionTime is the amount of time after which the minimum
	// required difficulty should be reduced when a block hasn't been found.
	//
	// NOTE: This only applies if ReduceMinDifficulty is true.
	MinDiffReductionTime time.Duration

	// GenerateSupported specifies whether or not CPU mining is allowed.
	GenerateSupported bool

	// MaximumBlockSizes are the maximum sizes of a block that can be
	// generated on the network.  It is an array because the max block size
	// can be different values depending on the results of a voting agenda.
	// The first entry is the initial block size for the network, while the
	// other entries are potential block size changes which take effect when
	// the vote for the associated agenda succeeds.
	MaximumBlockSizes []int

	// MaxTxSize is the maximum number of bytes a serialized transaction can
	// be in order to be considered valid by consensus.
	MaxTxSize int

	// Subsidy parameters.
	//
	// Subsidy calculation for exponential reductions:
	// 0 for i in range (0, height / SubsidyReductionInterval):
	// 1     subsidy *= MulSubsidy
	// 2     subsidy /= DivSubsidy
	//
	// Caveat: Don't overflow the int64 register!!

	// BaseSubsidy is the starting subsidy amount for mined blocks.
	BaseSubsidy int64

	// Subsidy reduction multiplier.
	MulSubsidy int64

	// Subsidy reduction divisor.
	DivSubsidy int64

	// SubsidyReductionInterval is the reduction interval in blocks.
	SubsidyReductionInterval int64

	// TargetTotalSubsidy is the target total subsidy.
	TargetTotalSubsidy int64

	// WorkRewardProportion is the comparative amount of the subsidy given for
	// creating a block.
	WorkRewardProportion uint16

	// StakeRewardProportion is the comparative amount of the subsidy given for
	// casting stake votes (collectively, per block).
	StakeRewardProportion uint16

	// BlockTaxProportion is the inverse of the percentage of funds for each
	// block to allocate to the developer organization.
	// e.g. 10% --> 10 (or 1 / (1/10))
	// Special case: disable taxes with a value of 0
	BlockTaxProportion uint16

	// It must be hourglass block.
	// Checkpoints ordered from oldest to newest.
	Checkpoints []Checkpoint

	// Mempool parameters
	RelayNonStdTxs bool

	// NetworkAddressPrefix is the first letter of the network
	// for any given address encoded as a string.
	NetworkAddressPrefix string

	// Human-readable part for Bech32 encoded segwit addresses
	Bech32HRPSegwit string

	// Address encoding magics
	PubKeyAddrID     [2]byte // First 2 bytes of a P2PK address
	PubKeyHashAddrID [2]byte // First 2 bytes of P2PKH address
	PKHEdwardsAddrID [2]byte // First 2 bytes of Edwards P2PKH address
	PKHSchnorrAddrID [2]byte // First 2 bytes of secp256k1 Schnorr P2PKH address

	ScriptHashAddrID [2]byte // First 2 bytes of a P2SH address
	PrivateKeyID     [2]byte // First 2 bytes of a WIF private key

	// BIP32 hierarchical deterministic extended key magics
	HDPrivateKeyID [4]byte
	HDPublicKeyID  [4]byte

	// SLIP-0044 is the registered coin type used for BIP44 coin type used in the
	// hierarchical deterministic path for address generation.
	// The SLIP-0044 coin type for qitmeer network is 813
	// https://github.com/satoshilabs/slips/blob/master/slip-0044.md
	SLIP0044CoinType uint32

	// Legacy BIP44 coin type used in the hierarchical deterministic path for
	// address generation.
	// - Previous name was HDCoinType, the LegacyCoinType is used for the backwards compatibility.
	// - SLIP0044CoinType should be used instead.
	LegacyCoinType uint32

	// OrganizationPkScript is the output script for block taxes to be
	// distributed to in every block's coinbase. It should ideally be a P2SH
	// multisignature address.
	// TODO revisit the org-pkscript design
	OrganizationPkScript []byte

	// TokenAdminPkScript is the output script for token
	// It should ideally be a P2SH multisignature address.
	TokenAdminPkScript []byte

	// the output script for guard lock address
	GuardAddrPkScript []byte
	// the output script for honor lock address
	HonorAddrPkScript []byte

	// DAG
	BlockDelay    float64
	BlockRate     float64
	SecurityLevel float64

	LedgerParams ledger.LedgerParams

	CoinbaseConfig CoinbaseConfigs

	// evm
	MeerConfig *eparams.ChainConfig

	AmanaConfig *eparams.ChainConfig

	MeerEVMForkBlock    *big.Int // MeerEVM is enabled  and new subsidy calculation
	MeerUTXOForkBlock   *big.Int // What main height can transfer the locked utxo in genesis to MeerEVM, Must after MeerEVMForkMainHeight
	EmptyBlockForkBlock *big.Int // main height
	GasLimitForkBlock   *big.Int // gaslimit fork meerevm block number
	CancunForkBlock     *big.Int // use custom diff when cancun fork
	MeerChangeForkBlock *big.Int // MeerChange system contract
}

type CoinbaseConfig struct {
	Height                    int64
	Version                   string
	ExtraDataIncludedVer      bool
	ExtraDataIncludedNodeInfo bool
}

type CoinbaseConfigs []CoinbaseConfig

func (cf *CoinbaseConfigs) CheckVersion(curHeight int64, coinbase []byte) bool {
	version := cf.GetCurrentVersion(curHeight)
	return version == "" || strings.Contains(string(coinbase), version)
}

func (cf *CoinbaseConfigs) GetCurrentVersion(curHeight int64) string {
	current := cf.GetCurrentConfig(curHeight)
	if current != nil && current.ExtraDataIncludedVer {
		return current.Version
	}
	return ""
}

func (cf *CoinbaseConfigs) GetCurrentConfig(curHeight int64) *CoinbaseConfig {
	var cc *CoinbaseConfig = nil
	for i := 0; i < len(*cf); i++ {
		config := (*cf)[i]
		if config.Height > curHeight {
			break
		}
		cc = &config
	}
	return cc
}

// TotalSubsidyProportions is the sum of POW Reward, POS Reward, and Tax
// proportions.
func (p *Params) TotalSubsidyProportions() uint16 {
	return p.WorkRewardProportion + p.StakeRewardProportion + p.BlockTaxProportion
}

// has tax
func (p *Params) HasTax() bool {
	if p.BlockTaxProportion > 0 &&
		len(p.OrganizationPkScript) > 0 {
		return true
	}
	return false
}

func (p *Params) IsDevelopDiff() bool {
	return p.PowConfig.DifficultyMode == pow.DIFFICULTY_MODE_DEVELOP
}

func (p *Params) IsMeerEVMFork(height int64) bool {
	return isBlockForked(p.MeerEVMForkBlock, big.NewInt(height))
}

func (p *Params) IsMeerUTXOFork(height int64) bool {
	return isBlockForked(p.MeerUTXOForkBlock, big.NewInt(height))
}

func (p *Params) IsEmptyBlockFork(height int64) bool {
	return isBlockForked(p.EmptyBlockForkBlock, big.NewInt(height))
}

func (p *Params) IsGasLimitFork(number *big.Int) bool {
	return isBlockForked(p.GasLimitForkBlock, number)
}

func (p *Params) IsCancunFork(number *big.Int) bool {
	return isBlockForked(p.CancunForkBlock, number)
}

func (p *Params) IsMeerChangeFork(number *big.Int) bool {
	return isBlockForked(p.MeerChangeForkBlock, number)
}

var (
	// ErrDuplicateNet describes an error where the parameters for a network
	// could not be set due to the network already being a standard
	// network or previously-registered into this package.
	ErrDuplicateNet = errors.New("duplicate network")

	// ErrUnknownHDKeyID describes an error where the provided id which
	// is intended to identify the network for a hierarchical deterministic
	// private extended key is not registered.
	ErrUnknownHDKeyID = errors.New("unknown hd private extended key bytes")
)

var (
	registeredNets       = make(map[protocol.Network]struct{})
	pubKeyHashAddrIDs    = make(map[[2]byte]struct{})
	scriptHashAddrIDs    = make(map[[2]byte]struct{})
	hdPrivToPubKeyIDs    = make(map[[4]byte][]byte)
	bech32SegwitPrefixes = make(map[string]struct{})
)

// Register registers the network parameters for a Bitcoin network.  This may
// error with ErrDuplicateNet if the network is already registered (either
// due to a previous Register call, or the network being one of the default
// networks).
//
// Network parameters should be registered into this package by a main package
// as early as possible.  Then, library packages may lookup networks or network
// parameters based on inputs and work regardless of the network being standard
// or not.
func Register(params *Params) error {
	if _, ok := registeredNets[params.Net]; ok {
		return ErrDuplicateNet
	}
	registeredNets[params.Net] = struct{}{}
	pubKeyHashAddrIDs[params.PubKeyHashAddrID] = struct{}{}
	scriptHashAddrIDs[params.ScriptHashAddrID] = struct{}{}
	hdPrivToPubKeyIDs[params.HDPrivateKeyID] = params.HDPublicKeyID[:]

	// A valid Bech32 encoded segwit address always has as prefix the
	// human-readable part for the given net followed by '1'.
	bech32SegwitPrefixes[params.Bech32HRPSegwit+"1"] = struct{}{}
	return nil
}

// mustRegister performs the same function as Register except it panics if there
// is an error.  This should only be called from package init functions.
func mustRegister(params *Params) {
	if err := Register(params); err != nil {
		panic("failed to register network: " + err.Error())
	}
}

func init() {
	// Register all default networks when the package is initialized.
	mustRegister(&MainNetParams)
	mustRegister(&TestNetParams)
	mustRegister(&PrivNetParams)
	mustRegister(&MixNetParams)
}

// TODO, move to hex util
func hexMustDecode(hexStr string) []byte {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	return b
}

// IsBech32SegwitPrefix returns whether the prefix is a known prefix for segwit
// addresses on any default or registered network.  This is used when decoding
// an address string into a specific address type.
func IsBech32SegwitPrefix(prefix string) bool {
	prefix = strings.ToLower(prefix)
	_, ok := bech32SegwitPrefixes[prefix]
	return ok
}

// isBlockForked returns whether a fork scheduled at block s is active at the
// given head block.
func isBlockForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func newUint64(val uint64) *uint64 { return &val }

// Amana
var amanaChainConfig = &eparams.ChainConfig{
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
	ShanghaiTime:        newUint64(0),
	CancunTime:          newUint64(0),
	Clique: &eparams.CliqueConfig{
		Period: 3,
		Epoch:  100,
	},
}
