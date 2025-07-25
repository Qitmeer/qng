/*
 * Copyright (c) 2017-2025 The qitmeer developers
 */

package poa

import (
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/v2/common/hash"
	"github.com/Qitmeer/qng/v2/consensus/engine/config"
	ptypes "github.com/Qitmeer/qng/v2/consensus/engine/poa/types"
	"github.com/Qitmeer/qng/v2/consensus/model"
	qtypes "github.com/Qitmeer/qng/v2/core/types"
	"github.com/Qitmeer/qng/v2/rpc/api"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	econsensus "github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/params"
)

const (
	checkpointInterval = 1024 // height of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Number of recent vote snapshots to keep in memory
	inmemorySignatures = 4096 // Number of recent block signatures to keep in memory

	wiggleTime = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

// Amana proof-of-authority protocol constants.
var (
	epochLength = uint64(30000) // Default number of blocks after which to checkpoint and reset the pending votes

	diffInTurn = uint32(2) // Block difficulty for in-turn signatures
	diffNoTurn = uint32(1) // Block difficulty for out-of-turn signatures
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte signature suffix missing")

	// errMismatchingCheckpointSigners is returned if a checkpoint block contains a
	// list of signers different than the one the local node calculated.
	errMismatchingCheckpointSigners = errors.New("mismatching signer list on checkpoint block")

	// errInvalidDifficulty is returned if the difficulty of a block neither 1 or 2.
	errInvalidDifficulty = errors.New("invalid difficulty")

	// errWrongDifficulty is returned if the difficulty of a block doesn't match the
	// turn of the signer.
	errWrongDifficulty = errors.New("wrong difficulty")

	// errInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	errInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	// errRecentlySigned is returned if a header is signed by an authorized entity
	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")
)

// SignerFn hashes and signs the data to be signed by a backing account.
type SignerFn func(signer accounts.Account, mimeType string, message []byte) ([]byte, error)

// DagPOA is the proof-of-authority and MeerDAG consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type DagPoA struct {
	config *config.PoAConfig // Consensus engine configuration parameters
	db     model.DataBase    // Database to store and retrieve snapshot checkpoints

	recents    *lru.Cache[hash.Hash, *Snapshot] // Snapshots for recent block to speed up reorgs
	signatures *sigLRU                          // Signatures of recent blocks to speed up mining

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer and proposals fields

	// The fields below are for testing only
	fakeDiff bool // Skip difficulty verifications
}

func New(conf *config.PoAConfig, db model.DataBase) *DagPoA {
	// Set any missing consensus parameters to their defaults
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	recents := lru.NewCache[hash.Hash, *Snapshot](inmemorySnapshots)
	signatures := lru.NewCache[hash.Hash, common.Address](inmemorySignatures)

	return &DagPoA{
		config:     conf,
		db:         db,
		recents:    recents,
		signatures: signatures,
		proposals:  make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *DagPoA) Author(header *qtypes.BlockHeader) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *DagPoA) VerifyHeader(block *qtypes.SerializedBlock) error {
	b, err := NewBlock(block)
	if err != nil {
		return err
	}
	hpoa := b.Header().Engine.(*ptypes.PoA)
	// Don't waste time checking blocks from the future
	if b.Header().Timestamp.Unix() > time.Now().Unix()+int64(c.config.Period) {
		return econsensus.ErrFutureBlock
	}
	// Checkpoint blocks need to enforce zero beneficiary
	checkpoint := (b.height % c.config.Epoch) == 0
	if checkpoint && hpoa.Beneficiary != (common.Address{}) {
		return errInvalidCheckpointBeneficiary
	}
	if !hpoa.VerifyVote() {
		return errInvalidVote
	}
	if checkpoint && hpoa.IsAuthorize() {
		return errInvalidCheckpointVote
	}

	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if b.height > 0 {
		if b.Header().Difficulty != diffInTurn && b.Header().Difficulty != diffNoTurn {
			return errInvalidDifficulty
		}
	}
	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(b)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *DagPoA) verifyCascadingFields(block *Block) error {
	// The genesis block is the always valid dead-end
	if block.height == 0 {
		return nil
	}
	p, err := c.db.GetBlock(block.parent())
	if err != nil {
		return err
	}
	parent, err := NewBlock(p)
	if err != nil {
		return err
	}
	// Ensure that the block's timestamp isn't too close to its parent
	if parent == nil || parent.height != block.height-1 || !parent.block.Hash().IsEqual(block.parent()) {
		return econsensus.ErrUnknownAncestor
	}
	if parent.Header().Timestamp.Unix()+int64(c.config.Period) > block.Header().Timestamp.Unix() {
		return errInvalidTimestamp
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(block.height-1, block.parent())
	if err != nil {
		return err
	}
	// If the block is a checkpoint block, verify the signer list
	if block.height%c.config.Epoch == 0 {
		if !block.Engine().IsSignersEqual(snap.signers()) {
			return errMismatchingCheckpointSigners
		}
	}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(snap, block)
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (c *DagPoA) snapshot(height uint64, hash *hash.Hash) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		blocks []*Block
		snap   *Snapshot
	)
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(*hash); ok {
			snap = s
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if height%checkpointInterval == 0 {
			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting snapshot from disk", "height", height, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at the genesis, snapshot the initial state. Alternatively if we're
		// at a checkpoint block without a parent (light client CHT), or we have piled
		// up more headers than allowed to be reorged (chain reinit from a freezer),
		// consider the checkpoint trusted and snapshot it.
		if height == 0 ||
			(height%c.config.Epoch == 0 && (len(blocks) > params.FullImmutabilityThreshold)) {
			b, err := c.db.GetBlock(hash)
			if err != nil {
				return nil, err
			}
			checkpoint, err := NewBlock(b)
			if err != nil {
				return nil, err
			}
			if checkpoint != nil {
				signers := checkpoint.Engine().Signers
				snap = newSnapshot(c.config, c.signatures, height, hash, signers)
				if err := snap.store(c.db); err != nil {
					return nil, err
				}
				log.Info("Stored checkpoint snapshot to disk", "height", height, "hash", checkpoint.block.Hash())
				break
			}
		}
		// No explicit parents (or no more left), reach out to the database
		b, err := c.db.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, fmt.Errorf("Can't find block:%s", hash.String())
		}
		block, err := NewBlock(b)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		height, hash = height-1, block.parent()
	}
	for i := 0; i < len(blocks)/2; i++ {
		blocks[i], blocks[len(blocks)-1-i] = blocks[len(blocks)-1-i], blocks[i]
	}
	snap, err := snap.apply(blocks)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Height%checkpointInterval == 0 && len(blocks) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "height", snap.Height, "hash", snap.Hash)
	}
	return snap, err
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *DagPoA) verifySeal(snap *Snapshot, block *Block) error {
	// Verifying the genesis block is not supported
	if block.height == 0 {
		return errUnknownBlock
	}
	// Resolve the authorization key and check against signers
	signer, err := ecrecover(block.Header(), c.signatures)
	if err != nil {
		return err
	}
	if _, ok := snap.Signers[signer]; !ok {
		return errUnauthorizedSigner
	}
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only fail if the current block doesn't shift it out
			if limit := uint64(len(snap.Signers)/2 + 1); seen > block.height-limit {
				return errRecentlySigned
			}
		}
	}
	// Ensure that the difficulty corresponds to the turn-ness of the signer
	if !c.fakeDiff {
		inturn := snap.inturn(block.height, signer)
		if inturn && block.Header().Difficulty != diffInTurn {
			return errWrongDifficulty
		}
		if !inturn && block.Header().Difficulty != diffNoTurn {
			return errWrongDifficulty
		}
	}
	return nil
}

func (c *DagPoA) Prepare(block *qtypes.Block) error {
	block.Header.Engine = ptypes.Empty()
	curBlock, err := NewBlock(qtypes.NewBlock(block))
	if err != nil {
		return err
	}
	// Assemble the voting snapshot to check which votes make sense
	snap, err := c.snapshot(curBlock.height-1, curBlock.parent())
	if err != nil {
		return err
	}
	c.lock.RLock()
	if curBlock.height%c.config.Epoch != 0 {
		// Gather all the proposals that make sense voting on
		addresses := make([]common.Address, 0, len(c.proposals))
		for address, authorize := range c.proposals {
			if snap.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		// If there's pending proposals, cast a vote on them
		if len(addresses) > 0 {
			curBlock.Engine().Beneficiary = addresses[rand.Intn(len(addresses))]
			if c.proposals[curBlock.Coinbase()] {
				curBlock.Engine().Authorize()
			} else {
				curBlock.Engine().Drop()
			}
		}
	}

	// Copy signer protected by mutex to avoid race condition
	signer := c.signer
	c.lock.RUnlock()

	// Set the correct difficulty
	curBlock.Header().Difficulty = calcDifficulty(snap, signer)

	// Ensure the extra data has all its components
	if curBlock.height%c.config.Epoch == 0 {
		curBlock.Engine().Signers = snap.signers()
	}

	// Ensure the timestamp has the correct delay
	parent, err := c.db.GetBlock(curBlock.parent())
	if err != nil {
		return err
	}
	if parent == nil {
		return econsensus.ErrUnknownAncestor
	}
	curBlock.Header().Timestamp = time.Unix(parent.Block().Header.Timestamp.Unix()+int64(c.config.Period), 0)
	if curBlock.Header().Timestamp.Unix() < time.Now().Unix() {
		curBlock.Header().Timestamp = time.Unix(time.Now().Unix(), 0)
	}
	return nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (c *DagPoA) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *DagPoA) Seal(block *qtypes.Block, results chan<- struct{}, stop <-chan struct{}) error {
	b, err := NewBlock(qtypes.NewBlock(block))
	if err != nil {
		return err
	}
	// Sealing the genesis block is not supported
	if b.height == 0 {
		return errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if c.config.Period == 0 && len(block.Transactions) <= 1 {
		return errors.New("sealing paused while waiting for transactions")
	}
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := c.snapshot(b.height-1, b.parent())
	if err != nil {
		return err
	}
	if _, authorized := snap.Signers[signer]; !authorized {
		return errUnauthorizedSigner
	}
	// If we're amongst the recent signers, wait for the next block
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only wait if the current block doesn't shift it out
			if limit := uint64(len(snap.Signers)/2 + 1); b.height < limit || seen > b.height-limit {
				return errors.New("signed recently, must wait for others")
			}
		}
	}
	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := b.Header().Timestamp.Sub(time.Now()) // nolint: gosimple
	if b.Header().Difficulty == diffNoTurn {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Signers)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle)))

		log.Trace("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypeClique, DagPoARLP(b.Header()))
	if err != nil {
		return err
	}
	b.Engine().Seal = sighash
	// Wait until sealing is terminated or delay timeout.
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}

		select {
		case results <- struct{}{}:
		case <-stop:
			log.Warn("Sealing result is not read by miner", "sealhash", SealHash(b.Header()))
		}
	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have:
// * DIFF_NOTURN(2) if BLOCK_NUMBER % SIGNER_COUNT != SIGNER_INDEX
// * DIFF_INTURN(1) if BLOCK_NUMBER % SIGNER_COUNT == SIGNER_INDEX
func (c *DagPoA) CalcDifficulty(time uint64, parent *Block) uint32 {
	snap, err := c.snapshot(parent.height, parent.block.Hash())
	if err != nil {
		return 0
	}
	c.lock.RLock()
	signer := c.signer
	c.lock.RUnlock()
	return calcDifficulty(snap, signer)
}

func calcDifficulty(snap *Snapshot, signer common.Address) uint32 {
	if snap.inturn(snap.Height+1, signer) {
		return diffInTurn
	}
	return diffNoTurn
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *DagPoA) SealHash(header *qtypes.BlockHeader) common.Hash {
	return SealHash(header)
}

func (c *DagPoA) Signer() common.Address {
	return c.signer
}

func (c *DagPoA) APIs(chain model.BlockChain) []api.API {
	return []api.API{
		{
			NameSpace: Identifier,
			Service:   NewPublicAPI(c, chain),
			Public:    true,
		},
	}
}
