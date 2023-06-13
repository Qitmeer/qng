// Copyright (c) 2017-2018 The qitmeer developers

package blockchain

import (
	"container/list"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/common/util"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain/token"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/dbnamespace"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/merkle"
	"github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/shutdown"
	"github.com/Qitmeer/qng/core/state"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/engine/txscript"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/progresslog"
	"github.com/schollz/progressbar/v3"
	"sort"
	"sync"
	"time"
)

const (

	// maxOrphanBlocks is the maximum number of orphan blocks that can be
	// queued.
	MaxOrphanBlocks = 500
	// minMemoryNodes is the minimum number of consecutive nodes needed
	// in memory in order to perform all necessary validation.  It is used
	// to determine when it's safe to prune nodes from memory without
	// causing constant dynamic reloading.  This value should be larger than
	// that for minMemoryStakeNodes.
	minMemoryNodes = 2880
)

// BlockChain provides functions such as rejecting duplicate blocks, ensuring
// blocks follow all rules, orphan handling, checkpoint handling, and best chain
// selection with reorganization.
type BlockChain struct {
	service.Service

	params *params.Params

	// The following fields are set when the instance is created and can't
	// be changed afterwards, so there is no need to protect them with a
	// separate mutex.
	checkpointsByLayer map[uint64]*params.Checkpoint

	db           database.DB
	dbInfo       *databaseInfo
	timeSource   model.MedianTimeSource
	events       *event.Feed
	sigCache     *txscript.SigCache
	indexManager model.IndexManager

	// subsidyCache is the cache that provides quick lookup of subsidy
	// values.
	subsidyCache *SubsidyCache

	// chainLock protects concurrent access to the vast majority of the
	// fields in this struct below this point.
	chainLock sync.RWMutex

	// These fields are configuration parameters that can be toggled at
	// runtime.  They are protected by the chain lock.
	noVerify      bool
	noCheckpoints bool

	// These fields are related to handling of orphan blocks.  They are
	// protected by a combination of the chain lock and the orphan lock.
	orphanLock   sync.RWMutex
	orphans      map[hash.Hash]*orphanBlock
	oldestOrphan *orphanBlock

	// These fields are related to checkpoint handling.  They are protected
	// by the chain lock.
	nextCheckpoint *params.Checkpoint
	checkpointNode meerdag.IBlock

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element

	// The state is used as a fairly efficient way to cache information
	// about the current best chain state that is returned to callers when
	// requested.  It operates on the principle of MVCC such that any time a
	// new block becomes the best block, the state pointer is replaced with
	// a new struct and the old state is left untouched.  In this way,
	// multiple callers can be pointing to different best chain states.
	// This is acceptable for most callers because the state is only being
	// queried at a specific point in time.
	//
	// In addition, some of the fields are stored in the database so the
	// chain state can be quickly reconstructed on load.
	stateLock     sync.RWMutex
	stateSnapshot *BestState

	// pruner is the automatic pruner for block nodes and stake nodes,
	// so that the memory may be restored by the garbage collector if
	// it is unlikely to be referenced in the future.
	pruner *chainPruner

	//block dag
	bd *meerdag.MeerDAG

	// cache notification
	CacheNotifications []*Notification

	notificationsLock sync.RWMutex
	notifications     []NotificationCallback

	// The ID of token state tip for the chain.
	TokenTipID uint32

	Acct model.Acct

	shutdownTracker *shutdown.Tracker

	consensus model.Consensus

	progressLogger *progresslog.BlockProgressLogger

	msgChan chan *processMsg
	wg      sync.WaitGroup
	quit    chan struct{}
}

func (b *BlockChain) Init() error {
	// Initialize the chain state from the passed database.  When the db
	// does not yet contain any chain state, both it and the chain state
	// will be initialized to contain only the genesis block.
	if err := b.initChainState(); err != nil {
		return err
	}
	// Initialize and catch up all of the currently active optional indexes
	// as needed.
	if b.indexManager != nil {
		err := b.indexManager.Init()
		if err != nil {
			return err
		}
	}
	b.pruner = newChainPruner(b)

	err := b.initCheckPoints()
	if err != nil {
		return err
	}

	//
	log.Info(fmt.Sprintf("DAG Type:%s", b.bd.GetName()))
	log.Info("Blockchain database version", "chain", b.dbInfo.version, "compression", b.dbInfo.compVer,
		"index", b.dbInfo.bidxVer)

	tips := b.bd.GetTipsList()
	log.Info(fmt.Sprintf("Chain state:totaltx=%d tipsNum=%d mainOrder=%d total=%d", b.BestSnapshot().TotalTxns, len(tips), b.bd.GetMainChainTip().GetOrder(), b.bd.GetBlockTotal()))

	for _, v := range tips {
		log.Info(fmt.Sprintf("hash=%s,order=%s,height=%d", v.GetHash(), meerdag.GetOrderLogStr(v.GetOrder()), v.GetHeight()))
	}
	return nil
}

// initChainState attempts to load and initialize the chain state from the
// database.  When the db does not yet contain any chain state, both it and the
// chain state are initialized to the genesis block.
func (b *BlockChain) initChainState() error {
	err := b.shutdownTracker.Check()
	if err != nil {
		return err
	}

	// Determine the state of the database.
	var isStateInitialized bool
	err = b.db.View(func(dbTx database.Tx) error {
		// Fetch the database versioning information.
		dbInfo, err := dbFetchDatabaseInfo(dbTx)
		if err != nil {
			return err
		}

		// The database bucket for the versioning information is missing.
		if dbInfo == nil {
			return nil
		}

		// Don't allow downgrades of the blockchain database.
		if dbInfo.version > currentDatabaseVersion {
			return fmt.Errorf("the current blockchain database is "+
				"no longer compatible with this version of "+
				"the software (%d > %d)", dbInfo.version,
				currentDatabaseVersion)
		}

		// Don't allow downgrades of the database compression version.
		if dbInfo.compVer > serialization.CurrentCompressionVersion {
			return fmt.Errorf("the current database compression "+
				"version is no longer compatible with this "+
				"version of the software (%d > %d)",
				dbInfo.compVer, serialization.CurrentCompressionVersion)
		}

		// Don't allow downgrades of the block index.
		if dbInfo.bidxVer > currentBlockIndexVersion {
			return fmt.Errorf("the current database block index "+
				"version is no longer compatible with this "+
				"version of the software (%d > %d)",
				dbInfo.bidxVer, currentBlockIndexVersion)
		}

		b.dbInfo = dbInfo
		isStateInitialized = true
		return nil
	})
	if err != nil {
		return err
	}

	// Initialize the database if it has not already been done.
	if !isStateInitialized {
		return b.createChainState()
	}

	//   Upgrade the database as needed.
	err = b.upgradeDB(b.consensus.Interrupt())
	if err != nil {
		return err
	}

	var state bestChainState
	// Attempt to load the chain state from the database.
	err = b.db.Update(func(dbTx database.Tx) error {
		// Fetch the stored chain state from the database metadata.
		// When it doesn't exist, it means the database hasn't been
		// initialized for use with chain yet, so break out now to allow
		// that to happen under a writable database transaction.
		meta := dbTx.Metadata()
		serializedData := meta.Get(dbnamespace.ChainStateKeyName)
		if serializedData == nil {
			return fmt.Errorf("No chain state data")
		}
		log.Trace("Serialized chain state: ", "serializedData", fmt.Sprintf("%x", serializedData))
		state, err = DeserializeBestChainState(serializedData)
		if err != nil {
			return err
		}
		log.Trace(fmt.Sprintf("Load chain state:%s %d %d %s %s", state.hash.String(), state.total, state.totalTxns, state.tokenTipHash.String(), state.workSum.Text(16)))
		return nil
	})
	if err != nil {
		return err
	}

	log.Info("Loading dag ...")
	bidxStart := roughtime.Now()

	err = b.bd.Load(uint(state.total), b.params.GenesisHash)
	if err != nil {
		return fmt.Errorf("The dag data was damaged (%s). you can cleanup your block data base by '--cleanup'.", err)
	}
	if !b.bd.GetMainChainTip().GetHash().IsEqual(&state.hash) {
		return fmt.Errorf("The dag main tip %s is not the same. %s", state.hash.String(), b.bd.GetMainChainTip().GetHash().String())
	}
	log.Info(fmt.Sprintf("Dag loaded:loadTime=%v", roughtime.Since(bidxStart)))
	// Set the best chain view to the stored best state.
	// Load the raw block bytes for the best block.
	mainTip := b.bd.GetMainChainTip()
	mainTipNode := b.GetBlockNode(mainTip)
	if mainTipNode == nil {
		return fmt.Errorf("No main tip")
	}

	var block *types.SerializedBlock
	err = b.db.View(func(dbTx database.Tx) error {
		block, err = dbFetchBlockByHash(dbTx, mainTip.GetHash())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Initialize the state related to the best block.
	blockSize := uint64(block.Block().SerializeSize())
	numTxns := uint64(len(block.Block().Transactions))

	b.TokenTipID = uint32(b.bd.GetBlockId(&state.tokenTipHash))
	b.stateSnapshot = newBestState(mainTip.GetHash(), mainTipNode.Difficulty(), blockSize, numTxns,
		b.CalcPastMedianTime(mainTip), state.totalTxns, b.bd.GetMainChainTip().GetState().GetWeight(),
		b.bd.GetGraphState(), &state.tokenTipHash, *mainTip.GetState().Root())
	ts := b.GetTokenState(b.TokenTipID)
	if ts == nil {
		return fmt.Errorf("token state error")
	}
	return ts.Commit()
}

// createChainState initializes both the database and the chain state to the
// genesis block.  This includes creating the necessary buckets and inserting
// the genesis block, so it must only be called on an uninitialized database.
func (b *BlockChain) createChainState() error {
	// Create a new node from the genesis block and set it as the best node.
	genesisBlock := types.NewBlock(b.params.GenesisBlock)
	genesisBlock.SetOrder(0)
	header := &genesisBlock.Block().Header
	node := NewBlockNode(genesisBlock)
	_, _, ib, _ := b.bd.AddBlock(node)
	ib.GetState().SetEVM(b.VMService().GetCurHeader())
	//node.FlushToDB(b)
	// Initialize the state related to the best block.  Since it is the
	// genesis block, use its timestamp for the median time.
	numTxns := uint64(len(genesisBlock.Block().Transactions))
	blockSize := uint64(genesisBlock.Block().SerializeSize())
	b.stateSnapshot = newBestState(node.GetHash(), node.Difficulty(), blockSize, numTxns,
		time.Unix(node.GetTimestamp(), 0), numTxns, 0, b.bd.GetGraphState(), node.GetHash(), *ib.GetState().Root())
	b.TokenTipID = 0
	// Create the initial the database chain state including creating the
	// necessary index buckets and inserting the genesis block.
	err := b.db.Update(func(dbTx database.Tx) error {
		meta := dbTx.Metadata()

		// Create the bucket that houses information about the database's
		// creation and version.
		_, err := meta.CreateBucket(dbnamespace.BCDBInfoBucketName)
		if err != nil {
			return err
		}

		b.dbInfo = &databaseInfo{
			version: currentDatabaseVersion,
			compVer: serialization.CurrentCompressionVersion,
			bidxVer: currentBlockIndexVersion,
			created: roughtime.Now(),
		}
		err = dbPutDatabaseInfo(dbTx, b.dbInfo)
		if err != nil {
			return err
		}

		// Create the bucket that houses the spend journal data.
		_, err = meta.CreateBucket(dbnamespace.SpendJournalBucketName)
		if err != nil {
			return err
		}

		// Create the bucket that houses the utxo set.  Note that the
		// genesis block coinbase transaction is intentionally not
		// inserted here since it is not spendable by consensus rules.
		_, err = meta.CreateBucket(dbnamespace.UtxoSetBucketName)
		if err != nil {
			return err
		}

		// Create the bucket which house the token state
		if _, err := meta.CreateBucket(dbnamespace.TokenBucketName); err != nil {
			return err
		}
		initTS := token.BuildGenesisTokenState()
		err = initTS.Commit()
		if err != nil {
			return err
		}
		err = token.DBPutTokenState(dbTx, uint32(ib.GetID()), initTS)
		if err != nil {
			return err
		}
		// Store the current best chain state into the database.
		err = dbPutBestState(dbTx, b.stateSnapshot, pow.CalcWork(header.Difficulty, header.Pow.GetPowType()))
		if err != nil {
			return err
		}

		// Add genesis utxo
		view := utxo.NewUtxoViewpoint()
		view.SetViewpoints([]*hash.Hash{genesisBlock.Hash()})
		for _, tx := range genesisBlock.Transactions() {
			view.AddTxOuts(tx, genesisBlock.Hash())
		}
		err = b.dbPutUtxoView(dbTx, view)
		if err != nil {
			return err
		}

		// Store the genesis block into the database.
		if err := dbTx.StoreBlock(genesisBlock); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return b.bd.Commit()
}

func (b *BlockChain) Start() error {
	if err := b.Service.Start(); err != nil {
		return err
	}
	b.wg.Add(1)
	go b.handler()

	// prepare evm env
	mainTip := b.bd.GetMainChainTip()
	evmHead, err := b.VMService().PrepareEnvironment(mainTip.GetState())
	if err != nil {
		return err
	}
	log.Info("prepare evm environment", "mainTipOrder", mainTip.GetOrder(), "mainTipHash", mainTip.GetHash().String(), "hash", evmHead.Hash().String(), "number", evmHead.Number.Uint64(), "root", evmHead.Root.String())
	return nil
}

func (b *BlockChain) Stop() error {
	if err := b.Service.Stop(); err != nil {
		return err
	}
	close(b.quit)
	b.wg.Wait()
	return nil
}

// HaveBlock returns whether or not the chain instance has the block represented
// by the passed hash.  This includes checking the various places a block can
// be like part of the main chain, on a side chain, or in the orphan pool.
//
// This function is safe for concurrent access.
func (b *BlockChain) HaveBlock(hash *hash.Hash) bool {
	return b.bd.HasBlock(hash) || b.IsOrphan(hash)
}

func (b *BlockChain) HasBlockInDB(h *hash.Hash) bool {
	err := b.db.View(func(dbTx database.Tx) error {
		has, er := dbTx.HasBlock(h)
		if er != nil {
			return er
		}
		if has {
			return nil
		}
		return fmt.Errorf("no")
	})
	return err == nil
}

// IsCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//   - Latest block height is after the latest checkpoint (if enabled)
//   - Latest block has a timestamp newer than 24 hours ago
//
// This function is safe for concurrent access.
func (b *BlockChain) IsCurrent() bool {
	b.ChainRLock()
	defer b.ChainRUnlock()
	return b.isCurrent()
}

// isCurrent returns whether or not the chain believes it is current.  Several
// factors are used to guess, but the key factors that allow the chain to
// believe it is current are:
//   - Latest block height is after the latest checkpoint (if enabled)
//   - Latest block has a timestamp newer than 24 hours ago
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockChain) isCurrent() bool {
	// Not current if the latest main (best) chain height is before the
	// latest known good checkpoint (when checkpoints are enabled).
	checkpoint := b.LatestCheckpoint()
	lastBlock := b.bd.GetMainChainTip()
	if checkpoint != nil && uint64(lastBlock.GetLayer()) < checkpoint.Layer {
		return false
	}
	// Not current if the latest best block has a timestamp before 24 hours
	// ago.
	//
	// The chain appears to be current if none of the checks reported
	// otherwise.
	minus24Hours := b.timeSource.AdjustedTime().Add(-24 * time.Hour).Unix()
	lastNode := b.GetBlockNode(lastBlock)
	if lastNode == nil {
		return false
	}
	return lastNode.GetTimestamp() >= minus24Hours
}

// TipGeneration returns the entire generation of blocks stemming from the
// parent of the current tip.
//
// The function is safe for concurrent access.
func (b *BlockChain) TipGeneration() ([]hash.Hash, error) {
	tips := b.bd.GetTipsList()
	tiphashs := []hash.Hash{}
	for _, block := range tips {
		tiphashs = append(tiphashs, *block.GetHash())
	}
	return tiphashs, nil
}

// BlockByHash returns the block from the main chain with the given hash.
//
// This function is safe for concurrent access.
func (b *BlockChain) BlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()

	return b.fetchMainChainBlockByHash(hash)
}

// HeaderByHash returns the block header identified by the given hash or an
// error if it doesn't exist.  Note that this will return headers from both the
// main chain and any side chains.
//
// This function is safe for concurrent access.
func (b *BlockChain) HeaderByHash(hash *hash.Hash) (types.BlockHeader, error) {
	block, err := b.fetchBlockByHash(hash)
	if err != nil || block == nil {
		return types.BlockHeader{}, fmt.Errorf("block %s is not known", hash)
	}

	return block.Block().Header, nil
}

// FetchBlockByHash searches the internal chain block stores and the database
// in an attempt to find the requested block.
//
// This function differs from BlockByHash in that this one also returns blocks
// that are not part of the main chain (if they are known).
//
// This function is safe for concurrent access.
func (b *BlockChain) FetchBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	return b.fetchBlockByHash(hash)
}

func (b *BlockChain) FetchBlockBytesByHash(hash *hash.Hash) ([]byte, error) {
	return b.fetchBlockBytesByHash(hash)
}

// fetchMainChainBlockByHash returns the block from the main chain with the
// given hash.  It first attempts to use cache and then falls back to loading it
// from the database.
//
// An error is returned if the block is either not found or not in the main
// chain.
//
// This function is safe for concurrent access.
func (b *BlockChain) fetchMainChainBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	if !b.MainChainHasBlock(hash) {
		return nil, fmt.Errorf("No block in main chain")
	}
	block, err := b.fetchBlockByHash(hash)
	return block, err
}

// MaximumBlockSize returns the maximum permitted block size for the block
// AFTER the given node.
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockChain) maxBlockSize() (int64, error) {

	maxSize := int64(b.params.MaximumBlockSizes[0])

	// The max block size is not changed in any other cases.
	return maxSize, nil
}

// fetchBlockByHash returns the block with the given hash from all known sources
// such as the internal caches and the database.
//
// This function is safe for concurrent access.
func (b *BlockChain) fetchBlockByHash(hash *hash.Hash) (*types.SerializedBlock, error) {
	// Check orphan cache.
	block := b.GetOrphan(hash)
	if block != nil {
		return block, nil
	}

	// Load the block from the database.
	dbErr := b.db.View(func(dbTx database.Tx) error {
		var err error
		block, err = dbFetchBlockByHash(dbTx, hash)
		return err
	})
	if dbErr == nil && block != nil {
		return block, nil
	}
	return nil, fmt.Errorf("unable to find block %v db", hash)
}

func (b *BlockChain) fetchBlockBytesByHash(hash *hash.Hash) ([]byte, error) {
	// Check orphan cache.
	block := b.GetOrphan(hash)
	if block != nil {
		return block.Bytes()
	}

	var bytes []byte
	var err error
	// Load the block from the database.
	err = b.db.View(func(dbTx database.Tx) error {
		bytes, err = dbTx.FetchBlock(hash)
		if err != nil {
			return err
		}
		return nil
	})
	return bytes, err
}

// FetchSubsidyCache returns the current subsidy cache from the blockchain.
//
// This function is safe for concurrent access.
func (b *BlockChain) FetchSubsidyCache() *SubsidyCache {
	return b.subsidyCache
}

// reorganizeChain reorganizes the block chain by disconnecting the nodes in the
// detachNodes list and connecting the nodes in the attach list.  It expects
// that the lists are already in the correct order and are in sync with the
// end of the current best chain.  Specifically, nodes that are being
// disconnected must be in reverse order (think of popping them off the end of
// the chain) and nodes the are being attached must be in forwards order
// (think pushing them onto the end of the chain).
//
// This function MUST be called with the chain state lock held (for writes).

func (b *BlockChain) reorganizeChain(ib meerdag.IBlock, detachNodes *list.List, attachNodes *list.List, newBlock *types.SerializedBlock, connectedBlocks *list.List) error {
	oldBlocks := []*hash.Hash{}
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		ob := e.Value.(*meerdag.BlockOrderHelp)
		oldBlocks = append(oldBlocks, ob.Block.GetHash())
	}

	b.sendNotification(Reorganization, &ReorganizationNotifyData{
		OldBlocks: oldBlocks,
		NewBlock:  newBlock.Hash(),
		NewOrder:  uint64(ib.GetOrder()),
	})

	// Why the old order is the order that was removed by the new block, because the new block
	// must be one of the tip of the dag.This is very important for the following understanding.
	// In the two case, the perspective is the same.In the other words, the future can not
	// affect the past.
	var block *types.SerializedBlock
	var err error

	for e := detachNodes.Back(); e != nil; e = e.Prev() {
		n := e.Value.(*meerdag.BlockOrderHelp)
		if n == nil {
			panic(fmt.Errorf("No BlockOrderHelp"))
		}
		b.updateTokenState(n.Block, nil, true)
		er := b.updateDefaultBlockState(n.Block)
		if er != nil {
			log.Error(er.Error())
		}
		//
		block, err = b.fetchBlockByHash(n.Block.GetHash())
		if err != nil {
			panic(err)
		}
		log.Debug("detach block", "hash", n.Block.GetHash().String(), "old order", n.OldOrder, "status", n.Block.GetState().GetStatus().String())
		block.SetOrder(uint64(n.OldOrder))
		// Load all of the utxos referenced by the block that aren't
		// already in the view.
		var stxos []utxo.SpentTxOut
		view := utxo.NewUtxoViewpoint()
		view.SetViewpoints([]*hash.Hash{block.Hash()})
		if !n.Block.GetState().GetStatus().KnownInvalid() {
			b.CalculateDAGDuplicateTxs(block)
			err = b.fetchInputUtxos(b.db, block, view)
			if err != nil {
				return err
			}

			// Load all of the spent txos for the block from the spend
			// journal.

			err = b.db.View(func(dbTx database.Tx) error {
				stxos, err = utxo.DBFetchSpendJournalEntry(dbTx, block)
				return err
			})
			if err != nil {
				return err
			}
			// Store the loaded block and spend journal entry for later.
			err = b.disconnectTransactions(block, stxos, view)
			if err != nil {
				b.bd.InvalidBlock(n.Block)
				log.Info(fmt.Sprintf("%s", err))
			}
		}
		b.bd.ValidBlock(n.Block)

		//newn.FlushToDB(b)

		err = b.disconnectBlock(n.Block, block, view, stxos)
		if err != nil {
			return err
		}
	}
	for e := attachNodes.Front(); e != nil; e = e.Next() {
		nodeBlock := e.Value.(meerdag.IBlock)
		if !nodeBlock.IsOrdered() {
			continue
		}
		startState := b.bd.GetBlockByOrder(nodeBlock.GetOrder() - 1).GetState()
		err = b.VMService().RewindTo(startState)
		if err != nil {
			return err
		}
		break
	}
	isEVMInit := false
	for e := attachNodes.Front(); e != nil; e = e.Next() {
		nodeBlock := e.Value.(meerdag.IBlock)
		if nodeBlock.GetID() == ib.GetID() {
			block = newBlock
		} else {
			// If any previous nodes in attachNodes failed validation,
			// mark this one as having an invalid ancestor.
			block, err = b.FetchBlockByHash(nodeBlock.GetHash())

			if err != nil {
				return err
			}
			block.SetOrder(uint64(nodeBlock.GetOrder()))
			block.SetHeight(nodeBlock.GetHeight())
		}
		if !nodeBlock.IsOrdered() {
			er := b.updateDefaultBlockState(nodeBlock)
			if er != nil {
				log.Error(er.Error())
			}
			continue
		}
		if !isEVMInit {
			isEVMInit = true
			err = b.prepareEVMEnvironment(nodeBlock)
			if err != nil {
				return err
			}
		}
		view := utxo.NewUtxoViewpoint()
		view.SetViewpoints([]*hash.Hash{nodeBlock.GetHash()})
		stxos := []utxo.SpentTxOut{}
		err = b.checkConnectBlock(nodeBlock, block, view, &stxos)
		if err != nil {
			b.bd.InvalidBlock(nodeBlock)
			stxos = []utxo.SpentTxOut{}
			view.Clean()
			log.Warn(err.Error(), "block", nodeBlock.GetHash().String(), "order", nodeBlock.GetOrder())
		}
		err = b.connectBlock(nodeBlock, block, view, stxos, connectedBlocks)
		if err != nil {
			b.bd.InvalidBlock(nodeBlock)
			er := b.updateDefaultBlockState(nodeBlock)
			if er != nil {
				log.Error(er.Error())
			}
			return err
		}
		if !nodeBlock.GetState().GetStatus().KnownInvalid() {
			b.bd.ValidBlock(nodeBlock)
		}
		b.bd.UpdateWeight(nodeBlock)
		er := b.updateBlockState(nodeBlock, block)
		if er != nil {
			log.Error(er.Error())
		}
		log.Debug("attach block", "hash", nodeBlock.GetHash().String(), "order", nodeBlock.GetOrder(), "status", nodeBlock.GetState().GetStatus().String())
	}

	// Log the point where the chain forked and old and new best chain
	// heads.
	log.Info(fmt.Sprintf("End DAG REORGANIZE: Old Len= %d;New Len= %d", detachNodes.Len(), attachNodes.Len()))

	return nil
}

// countSpentOutputs returns the number of utxos the passed block spends.
func (b *BlockChain) countSpentOutputs(block *types.SerializedBlock) int {
	// Exclude the coinbase transaction since it can't spend anything.
	var numSpent int
	for _, tx := range block.Transactions()[1:] {
		if tx.IsDuplicate {
			continue
		}
		if types.IsTokenTx(tx.Tx) {
			if types.IsTokenMintTx(tx.Tx) {
				numSpent--
			} else {
				continue
			}
		} else if types.IsCrossChainImportTx(tx.Tx) {
			numSpent++
			continue
		} else if types.IsCrossChainVMTx(tx.Tx) {
			continue
		}
		numSpent += len(tx.Transaction().TxIn)

	}
	return numSpent
}

// FetchSpendJournal can return the set of outputs spent for the target block.
func (b *BlockChain) FetchSpendJournal(targetBlock *types.SerializedBlock) ([]utxo.SpentTxOut, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()

	return b.fetchSpendJournal(targetBlock)
}

func (b *BlockChain) fetchSpendJournal(targetBlock *types.SerializedBlock) ([]utxo.SpentTxOut, error) {
	var spendEntries []utxo.SpentTxOut
	err := b.db.View(func(dbTx database.Tx) error {
		var err error

		spendEntries, err = utxo.DBFetchSpendJournalEntry(dbTx, targetBlock)
		return err
	})
	if err != nil {
		return nil, err
	}

	return spendEntries, nil
}

func (b *BlockChain) FetchSpendJournalPKS(targetBlock *types.SerializedBlock) ([][]byte, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()
	ret := [][]byte{}
	stxo, err := b.fetchSpendJournal(targetBlock)
	if err != nil {
		return nil, err
	}
	for _, so := range stxo {
		ret = append(ret, so.PkScript)
	}
	return ret, nil
}

func (b *BlockChain) ChainLock() {
	b.chainLock.Lock()
}

func (b *BlockChain) ChainUnlock() {
	b.chainLock.Unlock()
}

func (b *BlockChain) ChainRLock() {
	b.chainLock.RLock()
}

func (b *BlockChain) ChainRUnlock() {
	b.chainLock.RUnlock()
}

func (b *BlockChain) IsDuplicateTx(txid *hash.Hash, blockHash *hash.Hash) bool {
	err := b.db.Update(func(dbTx database.Tx) error {
		if b.indexManager != nil {
			if b.indexManager.IsDuplicateTx(dbTx, txid, blockHash) {
				return nil
			}
		}
		return fmt.Errorf("null")
	})
	return err == nil
}

func (b *BlockChain) CalculateDAGDuplicateTxs(block *types.SerializedBlock) {
	txs := block.Transactions()
	for _, tx := range txs {
		tx.IsDuplicate = b.IsDuplicateTx(tx.Hash(), block.Hash())
	}
}

func (b *BlockChain) CalculateFees(block *types.SerializedBlock) types.AmountMap {
	transactions := block.Transactions()
	totalAtomOut := types.AmountMap{}
	for i, tx := range transactions {
		if i == 0 || tx.Tx.IsCoinBase() || tx.IsDuplicate {
			continue
		}
		for k, txOut := range tx.Transaction().TxOut {
			if k == 0 && types.IsCrossChainExportTx(tx.Tx) {
				totalAtomOut[types.MEERA] += int64(txOut.Amount.Value)
			} else {
				totalAtomOut[txOut.Amount.Id] += int64(txOut.Amount.Value)
			}
		}
	}
	spentTxos, err := b.fetchSpendJournal(block)
	if err != nil {
		return nil
	}
	totalAtomIn := types.AmountMap{}
	if spentTxos != nil {
		for _, st := range spentTxos {
			if transactions[st.TxIndex].IsDuplicate {
				continue
			}
			totalAtomIn[st.Amount.Id] += int64(st.Amount.Value + st.Fees.Value)
		}

		totalFees := types.AmountMap{}
		for _, coinId := range types.CoinIDList {
			totalFees[coinId] = totalAtomIn[coinId] - totalAtomOut[coinId]
			if totalFees[coinId] < 0 {
				totalFees[coinId] = 0
			}
		}
		return totalFees
	}
	return nil
}

// GetFees
func (b *BlockChain) GetFees(h *hash.Hash) types.AmountMap {
	ib := b.GetBlock(h)
	if ib == nil {
		return nil
	}
	if ib.GetState().GetStatus().KnownInvalid() {
		return nil
	}
	block, err := b.FetchBlockByHash(h)
	if err != nil {
		return nil
	}
	b.CalculateDAGDuplicateTxs(block)

	return b.CalculateFees(block)
}

func (b *BlockChain) GetFeeByCoinID(h *hash.Hash, coinId types.CoinID) int64 {
	fees := b.GetFees(h)
	if fees == nil {
		return 0
	}
	return fees[coinId]
}

func (b *BlockChain) CalcWeight(ib meerdag.IBlock, bi *meerdag.BlueInfo) int64 {
	if ib.GetState().GetStatus().KnownInvalid() {
		return 0
	}
	block, err := b.FetchBlockByHash(ib.GetHash())
	if err != nil {
		log.Error(fmt.Sprintf("CalcWeight:%v", err))
		return 0
	}
	if b.IsDuplicateTx(block.Transactions()[0].Hash(), ib.GetHash()) {
		return 0
	}
	return b.subsidyCache.CalcBlockSubsidy(bi)
}

func (b *BlockChain) CalculateTokenStateRoot(txs []*types.Tx) *hash.Hash {
	updates := []token.ITokenUpdate{}
	for _, tx := range txs {
		if types.IsTokenTx(tx.Tx) {
			update, err := token.NewUpdateFromTx(tx.Tx)
			if err != nil {
				log.Error(err.Error())
				continue
			}
			updates = append(updates, update)
		}
	}
	if len(updates) <= 0 {
		return &hash.ZeroHash
	}
	balanceUpdate := []*hash.Hash{}
	for _, u := range updates {
		balanceUpdate = append(balanceUpdate, u.GetHash())
	}
	tsMerkle := merkle.BuildTokenBalanceMerkleTreeStore(balanceUpdate)

	return tsMerkle[0]
}

func (b *BlockChain) CalculateStateRoot(txs []*types.Tx) *hash.Hash {
	var vmGenesis *hash.Hash
	if b.VMService() != nil {
		vmGenesis = b.VMService().Genesis(txs)
	}
	tokenStateRoot := b.CalculateTokenStateRoot(txs)
	if tokenStateRoot.IsEqual(zeroHash) {
		if vmGenesis == nil || vmGenesis.IsEqual(zeroHash) {
			return &hash.ZeroHash
		}
		return vmGenesis
	} else {
		if vmGenesis == nil || vmGenesis.IsEqual(zeroHash) {
			return tokenStateRoot
		}
		return merkle.HashMerkleBranches(tokenStateRoot, vmGenesis)
	}
}

// CalcPastMedianTime calculates the median time of the previous few blocks
// prior to, and including, the block node.
//
// This function is safe for concurrent access.
func (b *BlockChain) CalcPastMedianTime(block meerdag.IBlock) time.Time {
	// Create a slice of the previous few block timestamps used to calculate
	// the median per the number defined by the constant medianTimeBlocks.
	timestamps := make([]int64, medianTimeBlocks)
	numNodes := 0
	iterBlock := block
	for i := 0; i < medianTimeBlocks && iterBlock != nil; i++ {
		iterNode := b.GetBlockNode(iterBlock)
		if iterNode == nil {
			break
		}
		timestamps[i] = iterNode.GetTimestamp()
		numNodes++

		iterBlock = b.bd.GetBlockById(iterBlock.GetMainParent())
	}

	// Prune the slice to the actual number of available timestamps which
	// will be fewer than desired near the beginning of the block chain
	// and sort them.
	timestamps = timestamps[:numNodes]
	sort.Sort(util.TimeSorter(timestamps))

	// NOTE: The consensus rules incorrectly calculate the median for even
	// numbers of blocks.  A true median averages the middle two elements
	// for a set with an even number of elements in it.   Since the constant
	// for the previous number of blocks to be used is odd, this is only an
	// issue for a few blocks near the beginning of the chain.  I suspect
	// this is an optimization even though the result is slightly wrong for
	// a few of the first blocks since after the first few blocks, there
	// will always be an odd number of blocks in the set per the constant.
	//
	// This code follows suit to ensure the same rules are used, however, be
	// aware that should the medianTimeBlocks constant ever be changed to an
	// even number, this code will be wrong.
	medianTimestamp := timestamps[numNodes/2]
	return time.Unix(medianTimestamp, 0)
}

// BestSnapshot returns information about the current best chain block and
// related state as of the current point in time.  The returned instance must be
// treated as immutable since it is shared by all callers.
//
// This function is safe for concurrent access.
func (b *BlockChain) BestSnapshot() *BestState {
	b.stateLock.RLock()
	snapshot := b.stateSnapshot
	b.stateLock.RUnlock()
	return snapshot
}

func (b *BlockChain) GetSubsidyCache() *SubsidyCache {
	return b.subsidyCache
}

func (b *BlockChain) DB() database.DB {
	return b.db
}

func (b *BlockChain) IndexManager() model.IndexManager {
	return b.indexManager
}

func (b *BlockChain) VMService() model.VMI {
	if b.consensus == nil {
		return nil
	}
	return b.consensus.VMService()
}

// Return chain params
func (b *BlockChain) ChainParams() *params.Params {
	return b.params
}

// Return the dag instance
func (b *BlockChain) BlockDAG() *meerdag.MeerDAG {
	return b.bd
}

// Return median time source
func (b *BlockChain) TimeSource() model.MedianTimeSource {
	return b.timeSource
}

func (b *BlockChain) Rebuild() error {
	b.TokenTipID = 0
	initTS := token.BuildGenesisTokenState()
	err := initTS.Commit()
	if err != nil {
		return err
	}
	gib := b.BlockDAG().GetBlockById(0)
	if gib == nil {
		return fmt.Errorf("No genesis block")
	}
	err = b.db.Update(func(tx database.Tx) error {
		err = token.DBPutTokenState(tx, uint32(gib.GetID()), initTS)
		if err != nil {
			return err
		}
		genesisBlock := types.NewBlock(params.ActiveNetParams.GenesisBlock)
		genesisBlock.SetOrder(0)

		view := utxo.NewUtxoViewpoint()
		view.SetViewpoints([]*hash.Hash{genesisBlock.Hash()})
		for _, tx := range genesisBlock.Transactions() {
			view.AddTxOuts(tx, genesisBlock.Hash())
		}
		err = b.dbPutUtxoView(tx, view)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	//
	logLvl := l.Glogger().GetVerbosity()
	bar := progressbar.Default(int64(b.GetMainOrder()), fmt.Sprintf("Rebuild:"))
	l.Glogger().Verbosity(l.LvlCrit)
	b.VMService().SetLogLevel(l.LvlCrit.String())
	defer func() {
		l.Glogger().Verbosity(logLvl)
		b.VMService().SetLogLevel(logLvl.String())
	}()

	var block *types.SerializedBlock
	for i := uint(0); i <= b.GetMainOrder(); i++ {
		bar.Add(1)
		if system.InterruptRequested(b.consensus.Interrupt()) {
			return fmt.Errorf("interrupt rebuild")
		}
		ib := b.bd.GetBlockByOrder(i)
		if ib == nil {
			return fmt.Errorf("No block order:%d", i)
		}
		err = nil
		block, err = b.fetchBlockByHash(ib.GetHash())
		if err != nil {
			return err
		}
		block.SetOrder(uint64(ib.GetOrder()))
		block.SetHeight(ib.GetHeight())

		if i == 0 {
			if b.indexManager != nil {
				err = b.indexManager.ConnectBlock(block, nil, ib)
				if err != nil {
					return err
				}
			}
			continue
		}

		view := utxo.NewUtxoViewpoint()
		view.SetViewpoints([]*hash.Hash{ib.GetHash()})
		stxos := []utxo.SpentTxOut{}
		err = b.checkConnectBlock(ib, block, view, &stxos)
		if err != nil {
			b.bd.InvalidBlock(ib)
			stxos = []utxo.SpentTxOut{}
			view.Clean()
		}
		connectedBlocks := list.New()
		err = b.connectBlock(ib, block, view, stxos, connectedBlocks)
		if err != nil {
			b.bd.InvalidBlock(ib)
			return err
		}
		if !ib.GetState().GetStatus().KnownInvalid() {
			b.bd.ValidBlock(ib)
		}
		b.bd.UpdateWeight(ib)
		err = b.bd.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BlockChain) GetBlockState(order uint64) model.BlockState {
	block := b.BlockDAG().GetBlockByOrder(uint(order))
	if block == nil {
		return nil
	}
	return block.GetState()
}

// New returns a BlockChain instance using the provided configuration details.
func New(consensus model.Consensus) (*BlockChain, error) {
	// Enforce required config fields.
	if consensus.DatabaseContext() == nil {
		return nil, model.AssertError("blockchain.New database is nil")
	}
	if consensus.Params() == nil {
		return nil, model.AssertError("blockchain.New chain parameters nil")
	}

	// Generate a checkpoint by height map from the provided checkpoints.
	par := consensus.Params()
	var checkpointsByLayer map[uint64]*params.Checkpoint
	var prevCheckpointLayer uint64
	if len(par.Checkpoints) > 0 {
		checkpointsByLayer = make(map[uint64]*params.Checkpoint)
		for i := range par.Checkpoints {
			checkpoint := &par.Checkpoints[i]
			if checkpoint.Layer <= prevCheckpointLayer {
				return nil, model.AssertError("blockchain.New " +
					"checkpoints are not sorted by height")
			}
			checkpointsByLayer[checkpoint.Layer] = checkpoint
			prevCheckpointLayer = checkpoint.Layer
		}
	}

	config := consensus.Config()
	b := BlockChain{
		consensus:          consensus,
		checkpointsByLayer: checkpointsByLayer,
		db:                 consensus.DatabaseContext(),
		params:             par,
		timeSource:         consensus.MedianTimeSource(),
		events:             consensus.Events(),
		sigCache:           consensus.SigCache(),
		indexManager:       consensus.IndexManager(),
		orphans:            make(map[hash.Hash]*orphanBlock),
		CacheNotifications: []*Notification{},
		shutdownTracker:    shutdown.NewTracker(config.DataDir),
		headerList:         list.New(),
		progressLogger:     progresslog.NewBlockProgressLogger("Processed", log),
		msgChan:            make(chan *processMsg),
		quit:               make(chan struct{}),
	}
	b.subsidyCache = NewSubsidyCache(0, b.params)

	b.bd = meerdag.New(config.DAGType, b.CalcWeight,
		1.0/float64(par.TargetTimePerBlock/time.Second),
		b.db, b.getBlockData, state.CreateBlockState, state.CreateBlockStateFromBytes)
	b.bd.SetTipsDisLimit(int64(par.CoinbaseMaturity))
	b.bd.SetCacheSize(config.DAGCacheSize, config.BlockDataCacheSize)

	b.InitServices()
	b.Services().RegisterService(b.bd)
	return &b, nil
}
