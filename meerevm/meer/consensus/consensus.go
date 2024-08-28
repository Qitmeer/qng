/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/consensus/forks"
	qtypes "github.com/Qitmeer/qng/core/types"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
	"math/big"
)

var (
	errUnclesUnsupported = errors.New("uncles unsupported")
	errOlderBlockTime    = errors.New("timestamp older than parent")
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
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			var err error
			if parent == nil {
				err = consensus.ErrUnknownAncestor
			} else {
				err = me.verifyHeader(chain, header, parent)
			}
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (me *MeerEngine) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errUnclesUnsupported
	}
	return nil
}

func (me *MeerEngine) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
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
		if !forks.NeedFixedGasLimit(parent.Number.Int64(), chain.Config().ChainID.Int64()) {
			if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
				return err
			}
		}
	} else if err := eip1559.VerifyEIP1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}

	// Verify existence / non-existence of withdrawalsHash.
	shanghai := chain.Config().IsShanghai(header.Number, header.Time)
	if shanghai && header.WithdrawalsHash == nil {
		return errors.New("missing withdrawalsHash")
	}
	if !shanghai && header.WithdrawalsHash != nil {
		return fmt.Errorf("invalid withdrawalsHash: have %x, expected nil", header.WithdrawalsHash)
	}
	// Verify the existence / non-existence of cancun-specific header fields
	cancun := chain.Config().IsCancun(header.Number, header.Time)
	if !cancun {
		switch {
		case header.ExcessBlobGas != nil:
			return fmt.Errorf("invalid excessBlobGas: have %d, expected nil", header.ExcessBlobGas)
		case header.BlobGasUsed != nil:
			return fmt.Errorf("invalid blobGasUsed: have %d, expected nil", header.BlobGasUsed)
		case header.ParentBeaconRoot != nil:
			return fmt.Errorf("invalid parentBeaconRoot, have %#x, expected nil", header.ParentBeaconRoot)
		}
	} else {
		if header.ParentBeaconRoot == nil {
			return errors.New("header is missing beaconRoot")
		}
		if err := eip4844.VerifyEIP4844Header(parent, header); err != nil {
			return err
		}
	}
	return nil
}

func (me *MeerEngine) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return forks.GetCancunForkDifficulty(parent.Number.Int64())
}

func (me *MeerEngine) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	number := header.Number.Int64()
	if number > 0 {
		number--
	}
	header.Difficulty = forks.GetCancunForkDifficulty(number)
	return nil
}

func (me *MeerEngine) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	me.OnExtraStateChange(chain, header, state)
	if me.StateChange != nil {
		me.StateChange(header, state, body)
	}
}

func (me *MeerEngine) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	shanghai := chain.Config().IsShanghai(header.Number, header.Time)
	if shanghai {
		// All blocks after Shanghai must include a withdrawals root.
		if body.Withdrawals == nil {
			body.Withdrawals = make([]*types.Withdrawal, 0)
		}
	} else {
		if len(body.Withdrawals) > 0 {
			return nil, errors.New("withdrawals set before Shanghai activation")
		}
	}
	// Finalize and assemble the block.
	me.Finalize(chain, header, state, body)

	// Assign the final state root to header.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Assemble and return the final block.
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
