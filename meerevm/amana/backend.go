package amana

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	qparams "github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"sync"
	"time"
)

const epochLength = 32

type withdrawalQueue struct {
	pending chan *types.Withdrawal
}

func (w *withdrawalQueue) add(withdrawal *types.Withdrawal) error {
	select {
	case w.pending <- withdrawal:
		break
	default:
		return errors.New("withdrawal queue full")
	}
	return nil
}

func (w *withdrawalQueue) gatherPending(maxCount int) []*types.Withdrawal {
	withdrawals := []*types.Withdrawal{}
	for {
		select {
		case withdrawal := <-w.pending:
			withdrawals = append(withdrawals, withdrawal)
			if len(withdrawals) == maxCount {
				return withdrawals
			}
		default:
			return withdrawals
		}
	}
}

type backend struct {
	shutdownCh  chan struct{}
	eth         *eth.Ethereum
	period      time.Duration
	withdrawals withdrawalQueue

	feeRecipient     common.Address
	feeRecipientLock sync.Mutex // lock gates concurrent access to the feeRecipient

	engineAPI          *catalyst.ConsensusAPI
	curForkchoiceState engine.ForkchoiceStateV1
	lastBlockTime      uint64
}

func NewBackend(period time.Duration, eth *eth.Ethereum) (*backend, error) {
	block := eth.BlockChain().CurrentBlock()
	current := engine.ForkchoiceStateV1{
		HeadBlockHash:      block.Hash(),
		SafeBlockHash:      block.Hash(),
		FinalizedBlockHash: block.Hash(),
	}
	engineAPI := catalyst.NewConsensusAPIQng(eth)

	return &backend{
		eth:                eth,
		period:             period,
		shutdownCh:         make(chan struct{}),
		engineAPI:          engineAPI,
		lastBlockTime:      block.Time,
		curForkchoiceState: current,
		withdrawals:        withdrawalQueue{make(chan *types.Withdrawal, 20)},
	}, nil
}

func (c *backend) SetFeeRecipient(feeRecipient common.Address) {
	c.feeRecipientLock.Lock()
	c.feeRecipient = feeRecipient
	c.feeRecipientLock.Unlock()
}

func (c *backend) Start() error {
	if c.period == 0 {
	} else {
		go c.loop()
	}
	return nil
}

func (c *backend) Stop() error {
	close(c.shutdownCh)
	return nil
}

func (c *backend) sealBlock(withdrawals []*types.Withdrawal, timestamp uint64) error {
	if timestamp <= c.lastBlockTime {
		timestamp = c.lastBlockTime + 1
	}
	c.feeRecipientLock.Lock()
	feeRecipient := c.feeRecipient
	c.feeRecipientLock.Unlock()

	if header := c.eth.BlockChain().CurrentBlock(); c.curForkchoiceState.HeadBlockHash != header.Hash() {
		finalizedHash := c.finalizedBlockHash(header.Number.Uint64())
		c.setCurrentState(header.Hash(), *finalizedHash)
	}

	var random [32]byte
	rand.Read(random[:])
	fcResponse, err := c.engineAPI.ForkchoiceUpdatedQng(c.curForkchoiceState, &engine.PayloadAttributes{
		Timestamp:             timestamp,
		SuggestedFeeRecipient: feeRecipient,
		Withdrawals:           nil,
		Random:                random,
		BeaconRoot:            nil,
	})
	if err != nil {
		return err
	}
	if fcResponse == engine.STATUS_SYNCING {
		return errors.New("chain rewind prevented invocation of payload creation")
	}
	envelope, err := c.engineAPI.GetPayloadQng(*fcResponse.PayloadID, true)
	if err != nil {
		return err
	}
	payload := envelope.ExecutionPayload
	block, err := engine.ExecutableDataToBlockQng(*payload, nil, nil)
	if err != nil {
		return err
	}
	resultCh := make(chan *types.Block)
	err = c.eth.Engine().Seal(c.eth.BlockChain(), block, resultCh, c.shutdownCh)
	if err != nil {
		return err
	}
	select {
	case result := <-resultCh:
		payload.ExtraData = result.Extra()
		payload.BlockHash = result.Hash()
	case <-c.shutdownCh:
		return fmt.Errorf("shutdown amana backend")
	}
	var finalizedHash common.Hash
	if payload.Number%epochLength == 0 {
		finalizedHash = payload.BlockHash
	} else {
		if fh := c.finalizedBlockHash(payload.Number); fh == nil {
			return errors.New("chain rewind interrupted calculation of finalized block hash")
		} else {
			finalizedHash = *fh
		}
	}

	// Independently calculate the blob hashes from sidecars.
	blobHashes := make([]common.Hash, 0)
	if envelope.BlobsBundle != nil {
		hasher := sha256.New()
		for _, commit := range envelope.BlobsBundle.Commitments {
			var c kzg4844.Commitment
			if len(commit) != len(c) {
				return errors.New("invalid commitment length")
			}
			copy(c[:], commit)
			blobHashes = append(blobHashes, kzg4844.CalcBlobHashV1(hasher, &c))
		}
	}
	// Mark the payload as canon
	if _, err = c.engineAPI.NewPayloadQng(*payload, nil, nil); err != nil {
		return err
	}
	c.setCurrentState(payload.BlockHash, finalizedHash)

	// Mark the block containing the payload as canonical
	if _, err = c.engineAPI.ForkchoiceUpdatedQng(c.curForkchoiceState, nil); err != nil {
		return err
	}
	c.lastBlockTime = payload.Timestamp

	// Broadcast the block and announce chain insertion event
	fb := c.eth.BlockChain().GetBlockByHash(c.eth.BlockChain().CurrentBlock().Hash())
	if fb != nil {
		c.eth.EventMux().Post(core.NewMinedBlockEvent{Block: fb})
	}
	return nil
}

func (c *backend) loop() {
	for {
		select {
		case <-c.shutdownCh:
			return
		default:
			withdrawals := c.withdrawals.gatherPending(10)
			if err := c.sealBlock(withdrawals, uint64(time.Now().Unix())); err != nil {
				log.Warn("Error performing sealing work", "err", err)
				time.Sleep(qparams.ActiveNetParams.TargetTimePerBlock)
			}
		}
	}
}

func (c *backend) finalizedBlockHash(number uint64) *common.Hash {
	var finalizedNumber uint64
	if number%epochLength == 0 {
		finalizedNumber = number
	} else {
		finalizedNumber = (number - 1) / epochLength * epochLength
	}
	if finalizedBlock := c.eth.BlockChain().GetBlockByNumber(finalizedNumber); finalizedBlock != nil {
		fh := finalizedBlock.Hash()
		return &fh
	}
	return nil
}

func (c *backend) setCurrentState(headHash, finalizedHash common.Hash) {
	c.curForkchoiceState = engine.ForkchoiceStateV1{
		HeadBlockHash:      headHash,
		SafeBlockHash:      headHash,
		FinalizedBlockHash: finalizedHash,
	}
}

// Commit seals a block on demand.
func (c *backend) Commit() common.Hash {
	withdrawals := c.withdrawals.gatherPending(10)
	if err := c.sealBlock(withdrawals, uint64(time.Now().Unix())); err != nil {
		log.Warn("Error performing sealing work", "err", err)
	}
	return c.eth.BlockChain().CurrentBlock().Hash()
}

func (c *backend) Rollback() {
	// Flush all transactions from the transaction pools
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(common.Big1, 256), common.Big1)
	c.eth.TxPool().SetGasTip(maxUint256)
	// Set the gas tip back to accept new transactions
	// TODO (Marius van der Wijden): set gas tip to parameter passed by config
	c.eth.TxPool().SetGasTip(big.NewInt(params.GWei))
}

// Fork sets the head to the provided hash.
func (c *backend) Fork(parentHash common.Hash) error {
	if len(c.eth.TxPool().Pending(txpool.PendingFilter{})) != 0 {
		return errors.New("pending block dirty")
	}
	parent := c.eth.BlockChain().GetBlockByHash(parentHash)
	if parent == nil {
		return errors.New("parent not found")
	}
	return c.eth.BlockChain().SetHead(parent.NumberU64())
}

// AdjustTime creates a new block with an adjusted timestamp.
func (c *backend) AdjustTime(adjustment time.Duration) error {
	if len(c.eth.TxPool().Pending(txpool.PendingFilter{})) != 0 {
		return errors.New("could not adjust time on non-empty block")
	}
	parent := c.eth.BlockChain().CurrentBlock()
	if parent == nil {
		return errors.New("parent not found")
	}
	withdrawals := c.withdrawals.gatherPending(10)
	return c.sealBlock(withdrawals, parent.Time+uint64(adjustment))
}
