/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"errors"
	"fmt"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
	"math/big"
	"runtime"
)

var (
	errUnclesUnsupported = errors.New("uncles unsupported")
)

func (me *MeerEngine) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

func (me *MeerEngine) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return me.verifyHeader(chain, header, parent)
}

func (me *MeerEngine) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	if len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = me.verifyHeaderWorker(chain, headers, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (me *MeerEngine) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return me.verifyHeader(chain, headers[index], parent)
}

func (me *MeerEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errUnclesUnsupported
	}
	return nil
}

func (me *MeerEngine) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify the block's gas usage and (if applicable) verify the base fee.
	if !chain.Config().IsLondon(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, expected 'nil'", header.BaseFee)
		}
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	return nil
}

func (me *MeerEngine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return big.NewInt(1)
}

func (me *MeerEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	header.Difficulty = big.NewInt(1)
	return nil
}

func (me *MeerEngine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	me.OnExtraStateChange(chain, header, state)
	if me.StateChange != nil {
		me.StateChange(header, state, body)
	}
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

func (me *MeerEngine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	me.Finalize(chain, header, state, body)

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)), nil
}

func (me *MeerEngine) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}

func (me *MeerEngine) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()
	header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
	select {
	case results <- block.WithSeal(header):
	default:
		me.log.Warn("Sealing result is not read by miner", "mode", "fake", "sealhash", me.SealHash(block.Header()))
	}
	return nil
}

func (me *MeerEngine) OnExtraStateChange(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB) {
	extdata := header.Extra
	if len(extdata) <= 1 {
		return
	}

	var tx = &types.Transaction{}
	if err := tx.UnmarshalBinary(extdata); err != nil {
		me.log.Error(fmt.Sprintf("rlp decoding failed: %v", err))
		return
	}
	oldBalance := state.GetBalance(*tx.To()).ToBig()
	if oldBalance == nil {
		oldBalance = big.NewInt(0)
	}

	if tx.Nonce() == uint64(qtypes.TxTypeCrossChainExport) {
		state.AddBalance(*tx.To(), uint256.MustFromBig(tx.Value()), tracing.BalanceChangeTransfer)
		me.log.Debug(fmt.Sprintf("Cross chain(%s):%s(MEER) => %s(ETH)", tx.To().String(), tx.Value().String(), tx.Value().String()))
	} else {
		fee := big.NewInt(0).Sub(oldBalance, tx.Value())
		if fee.Sign() < 0 {
			fee.SetInt64(0)
		}
		feeStr := ""
		if fee.Sign() > 0 {
			mfee := big.NewInt(0).Div(fee, qcommon.Precision)
			mfee = mfee.Mul(mfee, qcommon.Precision)
			efee := big.NewInt(0).Sub(fee, mfee)
			if efee.Sign() > 0 {
				feeStr = fmt.Sprintf("fee:%s(MEER)+%s(ETH)", mfee.String(), efee.String())
				state.AddBalance(header.Coinbase, uint256.MustFromBig(efee), tracing.BalanceIncreaseRewardMineBlock)
				nonce := state.GetNonce(header.Coinbase)
				state.SetNonce(header.Coinbase, nonce)
			} else {
				feeStr = fmt.Sprintf("fee:%s(MEER)", fee.String())
			}
		} else {
			feeStr = fmt.Sprintf("fee:%s(MEER)", fee.String())
		}
		state.SubBalance(*tx.To(), uint256.MustFromBig(oldBalance), tracing.BalanceChangeTransfer)
		me.log.Debug(fmt.Sprintf("Cross chain(%s):%s(ETH) => %s(MEER) + %s", tx.To().String(), oldBalance.String(), tx.Value().String(), feeStr))
	}

	newBalance := state.GetBalance(*tx.To()).ToBig()

	changeB := big.NewInt(0)
	changeB = changeB.Sub(newBalance, oldBalance)

	me.log.Debug(fmt.Sprintf("Balance(%s): %s => %s = %s", tx.To().String(), oldBalance.String(), newBalance.String(), changeB.String()))

	nonce := state.GetNonce(*tx.To())
	state.SetNonce(*tx.To(), nonce)
}
