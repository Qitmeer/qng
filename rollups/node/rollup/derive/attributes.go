package derive

import (
	"context"
	"fmt"

	"github.com/Qitmeer/qit/common"
	"github.com/Qitmeer/qit/common/hexutil"
	"github.com/Qitmeer/qit/core/types"

	"github.com/Qitmeer/qng/rollups/bindings/predeploys"
	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qng/rollups/node/rollup"
)

// L1ReceiptsFetcher fetches L1 header info and receipts for the payload attributes derivation (the info tx and deposits)
type L1ReceiptsFetcher interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
}

type SystemConfigL2Fetcher interface {
	SystemConfigByL2Hash(ctx context.Context, hash common.Hash) (eth.SystemConfig, error)
}

// FetchingAttributesBuilder fetches inputs for the building of L2 payload attributes on the fly.
type FetchingAttributesBuilder struct {
	cfg *rollup.Config
	l1  L1ReceiptsFetcher
	l2  SystemConfigL2Fetcher
}

func NewFetchingAttributesBuilder(cfg *rollup.Config, l1 L1ReceiptsFetcher, l2 SystemConfigL2Fetcher) *FetchingAttributesBuilder {
	return &FetchingAttributesBuilder{
		cfg: cfg,
		l1:  l1,
		l2:  l2,
	}
}

// PreparePayloadAttributes prepares a PayloadAttributes template that is ready to build a L2 block with deposits only, on top of the given l2Parent, with the given epoch as L1 origin.
// The template defaults to NoTxPool=true, and no sequencer transactions: the caller has to modify the template to add transactions,
// by setting NoTxPool=false as sequencer, or by appending batch transactions as verifier.
// The severity of the error is returned; a crit=false error means there was a temporary issue, like a failed RPC or time-out.
// A crit=true error means the input arguments are inconsistent or invalid.
func (ba *FetchingAttributesBuilder) PreparePayloadAttributes(ctx context.Context, l2Parent eth.L2BlockRef, epoch eth.BlockID) (attrs *eth.PayloadAttributes, err error) {
	var l1Info eth.BlockInfo
	var depositTxs []hexutil.Bytes
	var seqNumber uint64

	sysConfig, err := ba.l2.SystemConfigByL2Hash(ctx, l2Parent.Hash)
	if err != nil {
		return nil, NewTemporaryError(fmt.Errorf("failed to retrieve L2 parent block: %w", err))
	}

	// If the L1 origin changed this block, then we are in the first block of the epoch. In this
	// case we need to fetch all transaction receipts from the L1 origin block so we can scan for
	// user deposits.
	if l2Parent.L1Origin.Number != epoch.Number {
		info, receipts, err := ba.l1.FetchReceipts(ctx, epoch.Hash)
		if err != nil {
			return nil, NewTemporaryError(fmt.Errorf("failed to fetch L1 block info and receipts: %w", err))
		}
		if l2Parent.L1Origin.Hash != info.ParentHash() {
			return nil, NewResetError(
				fmt.Errorf("cannot create new block with L1 origin %s (parent %s) on top of L1 origin %s",
					epoch, info.ParentHash(), l2Parent.L1Origin))
		}

		deposits, err := DeriveDeposits(receipts, ba.cfg.DepositContractAddress)
		if err != nil {
			// deposits may never be ignored. Failing to process them is a critical error.
			return nil, NewCriticalError(fmt.Errorf("failed to derive some deposits: %w", err))
		}
		// apply sysCfg changes
		if err := UpdateSystemConfigWithL1Receipts(&sysConfig, receipts, ba.cfg); err != nil {
			return nil, NewCriticalError(fmt.Errorf("failed to apply derived L1 sysCfg updates: %w", err))
		}

		l1Info = info
		depositTxs = deposits
		seqNumber = 0
	} else {
		if l2Parent.L1Origin.Hash != epoch.Hash {
			return nil, NewResetError(fmt.Errorf("cannot create new block with L1 origin %s in conflict with L1 origin %s", epoch, l2Parent.L1Origin))
		}
		info, err := ba.l1.InfoByHash(ctx, epoch.Hash)
		if err != nil {
			return nil, NewTemporaryError(fmt.Errorf("failed to fetch L1 block info: %w", err))
		}
		l1Info = info
		depositTxs = nil
		seqNumber = l2Parent.SequenceNumber + 1
	}

	// Sanity check the L1 origin was correctly selected to maintain the time invariant between L1 and L2
	nextL2Time := l2Parent.Time + ba.cfg.BlockTime
	if nextL2Time < l1Info.Time() {
		return nil, NewResetError(fmt.Errorf("cannot build L2 block on top %s for time %d before L1 origin %s at time %d",
			l2Parent, nextL2Time, eth.ToBlockID(l1Info), l1Info.Time()))
	}

	l1InfoTx, err := L1InfoDepositBytes(seqNumber, l1Info, sysConfig, ba.cfg.IsRegolith(nextL2Time))
	if err != nil {
		return nil, NewCriticalError(fmt.Errorf("failed to create l1InfoTx: %w", err))
	}

	txs := make([]hexutil.Bytes, 0, 1+len(depositTxs))
	txs = append(txs, l1InfoTx)
	txs = append(txs, depositTxs...)

	return &eth.PayloadAttributes{
		Timestamp:             hexutil.Uint64(nextL2Time),
		PrevRandao:            eth.Bytes32(l1Info.MixDigest()),
		SuggestedFeeRecipient: predeploys.SequencerFeeVaultAddr,
		Transactions:          txs,
		NoTxPool:              true,
		GasLimit:              (*eth.Uint64Quantity)(&sysConfig.GasLimit),
	}, nil
}
