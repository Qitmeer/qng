// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/state"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/database/legacydb"
	"github.com/Qitmeer/qng/engine/txscript"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"time"
)

// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block chain.  It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block chain along with best chain selection and reorganization.
//
// When no errors occurred during processing, the first return value indicates
// the length of the fork the block extended.  In the case it either exteneded
// the best chain or is now the tip of the best chain due to causing a
// reorganize, the fork length will be 0.  The second return value indicates
// whether or not the block is an orphan, in which case the fork length will
// also be zero as expected, because it, by definition, does not connect ot the
// best chain.
//
// This function is safe for concurrent access.
// return IsOrphan,error
func (b *BlockChain) ProcessBlock(block *types.SerializedBlock, flags BehaviorFlags) (meerdag.IBlock, bool, error) {
	if b.IsShutdown() {
		return nil, false, fmt.Errorf("block chain is shutdown")
	}
	msg := processMsg{block: block, flags: flags, result: make(chan *processResult)}
	b.msgChan <- &msg
	result := <-msg.result
	return result.block, result.isOrphan, result.err
}

func (b *BlockChain) handler() {
	log.Trace("BlockChain handler")
out:
	for {
		select {
		case msg := <-b.msgChan:
			start := time.Now()
			ib, isOrphan, err := b.processBlock(msg.block, msg.flags)
			blockProcessTimer.Update(time.Since(start))
			msg.result <- &processResult{isOrphan: isOrphan, err: err, block: ib}
		case <-b.quit:
			break out
		}
	}

cleanup:
	for {
		select {
		case <-b.msgChan:
		default:
			break cleanup
		}
	}

	b.wg.Done()
	log.Trace("BlockChain handler done")
}

func (b *BlockChain) processBlock(block *types.SerializedBlock, flags BehaviorFlags) (meerdag.IBlock, bool, error) {
	isorphan, err := b.preProcessBlock(block, flags)
	if err != nil || isorphan {
		return nil, isorphan, err
	}
	// The block has passed all context independent checks and appears sane
	// enough to potentially accept it into the block chain.
	ib, err := b.maybeAcceptBlock(block, flags)
	if err != nil {
		return nil, false, err
	}
	// Accept any orphan blocks that depend on this block (they are no
	// longer orphans) and repeat for those accepted blocks until there are
	// no more.
	err = b.RefreshOrphans()
	if err != nil {
		return ib, false, err
	}

	log.Debug("Accepted block", "hash", block.Hash().String())
	return ib, false, nil
}

func (b *BlockChain) preProcessBlock(block *types.SerializedBlock, flags BehaviorFlags) (bool, error) {
	b.ChainRLock()
	defer b.ChainRUnlock()

	fastAdd := flags&BFFastAdd == BFFastAdd

	blockHash := block.Hash()
	log.Trace("Processing block ", "hash", blockHash)

	// The block must not already exist in the main chain or side chains.
	if b.bd.HasBlock(blockHash) {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, ruleError(ErrDuplicateBlock, str)
	}

	// The block must not already exist as an orphan.
	if b.IsOrphan(blockHash) {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return true, ruleError(ErrDuplicateBlock, str)
	}

	// Perform preliminary sanity checks on the block and its transactions.
	err := b.checkBlockSanity(block, b.timeSource, flags, b.params)
	if err != nil {
		return false, err
	}

	// Find the previous checkpoint and perform some additional checks based
	// on the checkpoint.  This provides a few nice properties such as
	// preventing old side chain blocks before the last checkpoint,
	// rejecting easy to mine, but otherwise bogus, blocks that could be
	// used to eat memory, and ensuring expected (versus claimed) proof of
	// work requirements since the previous checkpoint are met.
	blockHeader := &block.Block().Header
	checkpoint, err := b.findPreviousCheckpoint()
	if err != nil {
		return false, err
	}
	checkpointNode := b.GetBlockHeader(checkpoint)
	if checkpointNode != nil {
		// Ensure the block timestamp is after the checkpoint timestamp.
		checkpointTime := time.Unix(checkpointNode.Timestamp.Unix(), 0)
		if blockHeader.Timestamp.Before(checkpointTime) {
			str := fmt.Sprintf("block %v has timestamp %v before "+
				"last checkpoint timestamp %v", blockHash,
				blockHeader.Timestamp, checkpointTime)
			return false, ruleError(ErrCheckpointTimeTooOld, str)
		}

		if !fastAdd {
			// Even though the checks prior to now have already ensured the
			// proof of work exceeds the claimed amount, the claimed amount
			// is a field in the block header which could be forged.  This
			// check ensures the proof of work is at least the minimum
			// expected based on elapsed time since the last checkpoint and
			// maximum adjustment allowed by the retarget rules.
			duration := blockHeader.Timestamp.Sub(checkpointTime)
			requiredTarget := pow.CompactToBig(b.calcEasiestDifficulty(
				checkpointNode.Difficulty, duration, block.Block().Header.Pow))
			currentTarget := pow.CompactToBig(blockHeader.Difficulty)
			if !block.Block().Header.Pow.CompareDiff(currentTarget, requiredTarget) {
				str := fmt.Sprintf("block target difficulty of %064x "+
					"is too low when compared to the previous "+
					"checkpoint", currentTarget)
				return false, ruleError(ErrDifficultyTooLow, str)
			}
		}
	}

	// Handle orphan blocks.
	for _, pb := range block.Block().Parents {
		if !b.bd.HasBlock(pb) {
			log.Trace(fmt.Sprintf("Adding orphan block %s with parent %s", blockHash.String(), pb.String()))
			b.addOrphanBlock(block)

			// The fork length of orphans is unknown since they, by definition, do
			// not connect to the best chain.
			return true, nil
		}
	}
	return false, nil
}

// maybeAcceptBlock potentially accepts a block into the block chain and, if
// accepted, returns the length of the fork the block extended.  It performs
// several validation checks which depend on its position within the block chain
// before adding it.  The block is expected to have already gone through
// ProcessBlock before calling this function with it.  In the case the block
// extends the best chain or is now the tip of the best chain due to causing a
// reorganize, the fork length will be 0.
//
// The flags are also passed to checkBlockContext and connectBestChain.  See
// their documentation for how the flags modify their behavior.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) maybeAcceptBlock(block *types.SerializedBlock, flags BehaviorFlags) (meerdag.IBlock, error) {
	if onEnd := l.LogAndMeasureExecutionTime(log, "BlockChain.maybeAcceptBlock"); onEnd != nil {
		defer onEnd()
	}
	// This function should never be called with orphan blocks or the
	// genesis block.
	b.ChainLock()
	defer func() {
		b.flushNotifications()
	}()

	newNode := NewBlockNode(block)

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		mainParent := b.bd.GetBlock(newNode.GetMainParent())
		if mainParent == nil {
			b.ChainUnlock()
			return nil, fmt.Errorf("Can't find main parent")
		}
		// The block must pass all of the validation rules which depend on the
		// position of the block within the block chain.
		err := b.checkBlockContext(block, mainParent, flags)
		if err != nil {
			b.ChainUnlock()
			return nil, err
		}
	}

	// Prune stake nodes which are no longer needed before creating a new
	// node.
	b.pruner.pruneChainIfNeeded()

	//dag
	newOrders, oldOrders, ib, isMainChainTipChange := b.bd.AddBlock(newNode)
	if ib == nil {
		b.ChainUnlock()
		return nil, fmt.Errorf("Irreparable error![%s]", newNode.GetHash().String())
	}
	// Insert the block into the database if it's not already there.  Even
	// though it is possible the block will ultimately fail to connect, it
	// has already passed all proof-of-work and validity tests which means
	// it would be prohibitively expensive for an attacker to fill up the
	// disk with a bunch of blocks that fail to connect.  This is necessary
	// since it allows block download to be decoupled from the much more
	// expensive connection logic.  It also has some other nice properties
	// such as making blocks that never become part of the main chain or
	// blocks that fail to connect available for further analysis.
	//
	// Also, store the associated block index entry.
	err := b.db.Update(func(dbTx legacydb.Tx) error {
		exists, err := dbTx.HasBlock(block.Hash())
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		err = dbMaybeStoreBlock(dbTx, block)
		if err != nil {
			if legacydb.IsError(err, legacydb.ErrBlockExists) {
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		panic(err.Error())
	}
	err = b.shutdownTracker.Wait(ib.GetHash())
	if err != nil {
		panic(err.Error())
	}
	connectedBlocks := list.New()
	// Connect the passed block to the chain while respecting proper chain
	// selection according to the chain with the most proof of work.  This
	// also handles validation of the transaction scripts.
	_, err = b.connectDagChain(ib, newNode, newOrders, oldOrders, connectedBlocks)
	if err != nil {
		panic(err.Error())
	}
	err = b.updateBestState(ib, block, newOrders)
	if err != nil {
		panic(err.Error())
	}
	err = b.shutdownTracker.Done()
	if err != nil {
		panic(err.Error())
	}
	b.ChainUnlock()
	if connectedBlocks.Len() > 0 {
		for e := connectedBlocks.Front(); e != nil; e = e.Next() {
			b.sendNotification(BlockConnected, e.Value)
		}
	}

	if flags&BFP2PAdd == BFP2PAdd {
		b.progressLogger.LogBlockOrder(ib.GetOrder(), block)
	}

	// Notify the caller that the new block was accepted into the block
	// chain.  The caller would typically want to react by relaying the
	// inventory to other peers.
	b.sendNotification(BlockAccepted, &BlockAcceptedNotifyData{
		IsMainChainTipChange: isMainChainTipChange,
		Block:                block,
		Flags:                flags,
		Height:               uint64(ib.GetHeight()),
	})
	if b.Acct != nil {
		err = b.Acct.Commit()
		if err != nil {
			log.Error(err.Error())
		}
	}
	return ib, nil
}

func (b *BlockChain) FastAcceptBlock(block *types.SerializedBlock, flags BehaviorFlags) (meerdag.IBlock, error) {
	return b.maybeAcceptBlock(block, flags)
}

// connectBestChain handles connecting the passed block to the chain while
// respecting proper chain selection according to the chain with the most
// proof of work.  In the typical case, the new block simply extends the main
// chain.  However, it may also be extending (or creating) a side chain (fork)
// which may or may not end up becoming the main chain depending on which fork
// cumulatively has the most proof of work.  It returns the resulting fork
// length, that is to say the number of blocks to the fork point from the main
// chain, which will be zero if the block ends up on the main chain (either
// due to extending the main chain or causing a reorganization to become the
// main chain).
//
// The flags modify the behavior of this function as follows:
//   - BFFastAdd: Avoids several expensive transaction validation operations.
//     This is useful when using checkpoints.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) connectDagChain(ib meerdag.IBlock, block *BlockNode, newOrders *list.List, oldOrders *list.List, connectedBlocks *list.List) (bool, error) {

	if oldOrders.Len() <= 0 {
		newOr := []meerdag.IBlock{}
		for e := newOrders.Front(); e != nil; e = e.Next() {
			nodeBlock := e.Value.(meerdag.IBlock)
			newOr = append(newOr, nodeBlock)
		}
		if len(newOr) <= 0 {
			newOr = append(newOr, ib)
		}
		var sb *BlockNode
		var err error
		isEVMInit := false
		for _, nodeBlock := range newOr {
			if nodeBlock.GetID() == ib.GetID() {
				sb = block
			} else {
				sb = b.GetBlockNode(nodeBlock)
				if sb == nil {
					return false, fmt.Errorf("No block data:%s", nodeBlock.GetHash().String())
				}
			}

			if !nodeBlock.IsOrdered() {
				er := b.updateDefaultBlockState(nodeBlock)
				if er != nil {
					log.Error(er.Error())
				}
				continue
			}
			if sb == nil {
				return false, fmt.Errorf("No block:%s,id:%d\n", nodeBlock.GetHash().String(), nodeBlock.GetID())
			}
			if !isEVMInit {
				isEVMInit = true
				err = b.prepareEVMEnvironment(nodeBlock)
				if err != nil {
					return false, err
				}
			}
			view := utxo.NewUtxoViewpoint()
			view.SetViewpoints([]*hash.Hash{nodeBlock.GetHash()})
			stxos := []utxo.SpentTxOut{}
			err = b.checkConnectBlock(nodeBlock, sb, view, &stxos)
			if err != nil {
				b.bd.InvalidBlock(nodeBlock)
				stxos = []utxo.SpentTxOut{}
				view.Clean()
				log.Warn(err.Error(), "block", nodeBlock.GetHash().String(), "order", nodeBlock.GetOrder())
			}
			err = b.connectBlock(nodeBlock, sb, view, stxos, connectedBlocks)
			if err != nil {
				b.bd.InvalidBlock(nodeBlock)
				er := b.updateDefaultBlockState(nodeBlock)
				if er != nil {
					log.Error(er.Error())
				}
				return false, err
			}
			if !nodeBlock.GetState().GetStatus().KnownInvalid() {
				b.bd.ValidBlock(nodeBlock)
			}
			er := b.updateBlockState(nodeBlock, sb.GetBody())
			if er != nil {
				log.Error(er.Error())
			}
			log.Debug("Block connected to the main chain", "hash", nodeBlock.GetHash(), "order", nodeBlock.GetOrder())
		}
		return true, nil
	}

	// We're extending (or creating) a side chain and the cumulative work
	// for this new side chain is more than the old best chain, so this side
	// chain needs to become the main chain.  In order to accomplish that,
	// find the common ancestor of both sides of the fork, disconnect the
	// blocks that form the (now) old fork from the main chain, and attach
	// the blocks that form the new chain to the main chain starting at the
	// common ancenstor (the point where the chain forked).

	// Reorganize the chain.
	log.Info(fmt.Sprintf("Start DAG REORGANIZE: Block %v is causing a reorganize.", ib.GetHash()))
	err := b.reorganizeChain(ib, oldOrders, newOrders, block, connectedBlocks)
	if err != nil {
		return false, err
	}
	//b.updateBestState(node, block)
	return true, nil
}

// connectBlock handles connecting the passed node/block to the end of the main
// (best) chain.
//
// This passed utxo view must have all referenced txos the block spends marked
// as spent and all of the new txos the block creates added to it.  In addition,
// the passed stxos slice must be populated with all of the information for the
// spent txos.  This approach is used because the connection validation that
// must happen prior to calling this function requires the same details, so
// it would be inefficient to repeat it.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) connectBlock(node meerdag.IBlock, blockNode *BlockNode, view *utxo.UtxoViewpoint, stxos []utxo.SpentTxOut, connectedBlocks *list.List) error {
	block := blockNode.GetBody()
	pkss := [][]byte{}
	for _, stxo := range stxos {
		pkss = append(pkss, stxo.PkScript)
	}
	if !node.GetState().GetStatus().KnownInvalid() {
		_, err := b.meerConnectBlock(blockNode)
		if err != nil {
			return err
		}

		// Atomically insert info into the database.
		err = b.db.Update(func(dbTx legacydb.Tx) error {
			// Update the utxo set using the state of the utxo view.  This
			// entails removing all of the utxos spent and adding the new
			// ones created by the block.
			return b.dbPutUtxoView(dbTx, view)
		})
		if err != nil {
			return err
		}
		err = utxo.DBPutSpendJournalEntry(b.consensus.DatabaseContext(), block.Hash(), stxos)
		if err != nil {
			return err
		}
		// Allow the index manager to call each of the currently active
		// optional indexes with the block being connected so they can
		// update themselves accordingly.
		if b.indexManager != nil {
			err := b.indexManager.ConnectBlock(block, pkss, node)
			if err != nil {
				return fmt.Errorf("%v. (Attempt to execute --droptxindex)", err)
			}
		}

		// Prune fully spent entries and mark all entries in the view unmodified
		// now that the modifications have been committed to the database.
		view.Commit()

		err = b.updateTokenState(node, block, false)
		if err != nil {
			return err
		}
	} else {
		// Atomically insert info into the database.
		if b.indexManager != nil {
			err := b.indexManager.ConnectBlock(block, pkss, node)
			if err != nil {
				return err
			}
		}
	}
	connectedBlocks.PushBack([]interface{}{block, b.bd.IsOnMainChain(node.GetID()), node})
	return nil
}

// disconnectBlock handles disconnecting the passed node/block from the end of
// the main (best) chain.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) disconnectBlock(ib meerdag.IBlock, block *types.SerializedBlock, view *utxo.UtxoViewpoint, stxos []utxo.SpentTxOut) error {
	// Calculate the exact subsidy produced by adding the block.
	err := b.db.Update(func(dbTx legacydb.Tx) error {
		// Update the utxo set using the state of the utxo view.  This
		// entails restoring all of the utxos spent and removing the new
		// ones created by the block.
		return b.dbPutUtxoView(dbTx, view)
	})
	if err != nil {
		return err
	}
	err = utxo.DBRemoveSpendJournalEntry(b.consensus.DatabaseContext(), block.Hash())
	if err != nil {
		return err
	}
	// Allow the index manager to call each of the currently active
	// optional indexes with the block being disconnected so they
	// can update themselves accordingly.
	if b.indexManager != nil {
		pkss := [][]byte{}
		for _, stxo := range stxos {
			pkss = append(pkss, stxo.PkScript)
		}
		err := b.indexManager.DisconnectBlock(block, pkss, ib)
		if err != nil {
			return fmt.Errorf("%v. (Attempt to execute --droptxindex)", err)
		}
	}
	// Prune fully spent entries and mark all entries in the view unmodified
	// now that the modifications have been committed to the database.
	view.Commit()

	b.sendNotification(BlockDisconnected, []interface{}{block, ib})
	return nil
}

// connectTransaction updates the view by adding all new utxos created by the
// passed transaction and marking all utxos that the transactions spend as
// spent.  In addition, when the 'stxos' argument is not nil, it will be updated
// to append an entry for each spent txout.  An error will be returned if the
// view does not contain the required utxos.
func (bc *BlockChain) connectTransaction(tx *types.Tx, node *BlockNode, blockIndex uint32, stxos *[]utxo.SpentTxOut, view *utxo.UtxoViewpoint) error {
	msgTx := tx.Transaction()
	// Coinbase transactions don't have any inputs to spend.
	if msgTx.IsCoinBase() {
		// Add the transaction's outputs as available utxos.
		view.AddTxOuts(tx, node.GetHash()) //TODO, remove type conversion
		return nil
	}

	// Spend the referenced utxos by marking them spent in the view and,
	// if a slice was provided for the spent txout details, append an entry
	// to it.
	for txInIndex, txIn := range msgTx.TxIn {
		if txInIndex == 0 && types.IsTokenMintTx(tx.Tx) {
			continue
		}
		entry := view.Entries()[txIn.PreviousOut]

		// Ensure the referenced utxo exists in the view.  This should
		// never happen unless there is a bug is introduced in the code.
		if entry == nil {
			return model.AssertError(fmt.Sprintf("view missing input %v",
				txIn.PreviousOut))
		}
		entry.Spend()

		// Don't create the stxo details if not requested.
		if stxos == nil {
			continue
		}

		// Populate the stxo details using the utxo entry.  When the
		// transaction is fully spent, set the additional stxo fields
		// accordingly since those details will no longer be available
		// in the utxo set.
		var stxo = utxo.SpentTxOut{
			Amount:     entry.Amount(),
			Fees:       types.Amount{Value: 0, Id: entry.Amount().Id},
			PkScript:   entry.PkScript(),
			BlockHash:  *entry.BlockHash(),
			IsCoinBase: entry.IsCoinBase(),
			TxIndex:    uint32(tx.Index()),
			TxInIndex:  uint32(txInIndex),
		}
		if stxo.IsCoinBase && !entry.BlockHash().IsEqual(bc.params.GenesisHash) {
			if txIn.PreviousOut.OutIndex == CoinbaseOutput_subsidy ||
				entry.Amount().Id != types.MEERA {
				stxo.Fees.Value = bc.GetFeeByCoinID(&stxo.BlockHash, stxo.Fees.Id)
			}
		}
		// Append the entry to the provided spent txouts slice.
		*stxos = append(*stxos, stxo)
	}

	// Add the transaction's outputs as available utxos.
	view.AddTxOuts(tx, node.GetHash()) //TODO, remove type conversion

	return nil
}

func connectImportTransaction(tx *types.Tx, node *BlockNode, blockIndex uint32, stxos *[]utxo.SpentTxOut, balance int64, view *utxo.UtxoViewpoint) error {
	if stxos == nil {
		return nil
	}
	var stxo = utxo.SpentTxOut{
		Amount:     types.Amount{Id: types.MEERA, Value: balance},
		Fees:       types.Amount{Value: 0, Id: types.MEERA},
		PkScript:   nil,
		BlockHash:  hash.ZeroHash,
		IsCoinBase: false,
		TxIndex:    0,
		TxInIndex:  0,
	}
	*stxos = append(*stxos, stxo)
	view.AddTxOuts(tx, node.GetHash())
	return nil
}

// disconnectTransactions updates the view by removing all of the transactions
// created by the passed block, restoring all utxos the transactions spent by
// using the provided spent txo information, and setting the best hash for the
// view to the block before the passed block.
//
// This function will ONLY work correctly for a single transaction tree at a
// time because of index tracking.
func (bc *BlockChain) disconnectTransactions(block *types.SerializedBlock, stxos []utxo.SpentTxOut, view *utxo.UtxoViewpoint) error {
	// Sanity check the correct number of stxos are provided.
	if len(stxos) != bc.countSpentOutputs(block) {
		return model.AssertError("disconnectTransactions called with bad " +
			"spent transaction out information")
	}

	stxoIdx := len(stxos) - 1
	transactions := block.Transactions()
	for txIdx := len(transactions) - 1; txIdx > -1; txIdx-- {
		tx := transactions[txIdx]
		if tx.IsDuplicate {
			continue
		}
		if types.IsTokenTx(tx.Tx) {
			if !types.IsTokenMintTx(tx.Tx) {
				continue
			}
		} else if types.IsCrossChainVMTx(tx.Tx) {
			continue
		}
		isCoinBase := txIdx == 0
		txHash := tx.Hash()
		prevOut := types.TxOutPoint{Hash: *txHash}
		for txOutIdx, txOut := range tx.Tx.TxOut {
			if txscript.IsUnspendable(txOut.PkScript) {
				continue
			}

			prevOut.OutIndex = uint32(txOutIdx)
			entry := view.GetEntry(prevOut)
			if entry == nil {
				entry = utxo.NewUtxoEntry(txOut.Amount, txOut.PkScript, block.Hash(), isCoinBase)
				view.AddEntry(prevOut, entry)
			}

			entry.Spend()
		}

		if isCoinBase {
			continue
		} else if types.IsCrossChainImportTx(tx.Tx) {
			stxoIdx--
			continue
		}
		for txInIdx := len(tx.Tx.TxIn) - 1; txInIdx > -1; txInIdx-- {
			if types.IsTokenMintTx(tx.Tx) && txInIdx == 0 {
				continue
			}
			stxo := &stxos[stxoIdx]
			stxoIdx--

			originOut := &tx.Tx.TxIn[txInIdx].PreviousOut
			entry := view.GetEntry(*originOut)
			if entry == nil {
				entry = &utxo.UtxoEntry{}
				view.AddEntry(*originOut, entry)
			}
			entry.SetAmount(stxo.Amount)
			entry.SetPkScript(stxo.PkScript)
			entry.SetBlockHash(&stxo.BlockHash)
			entry.Modified()
			if stxo.IsCoinBase {
				entry.CoinBase()
			}
		}
	}

	view.SetViewpoints(nil)
	return nil
}

func (b *BlockChain) updateBestState(ib meerdag.IBlock, block *types.SerializedBlock, attachNodes *list.List) error {
	// Must be end node of sequence in dag
	// Generate a new best state snapshot that will be used to update the
	// database and later memory if all database updates are successful.
	lastState := b.BestSnapshot()

	// Calculate the number of transactions that would be added by adding
	// this block.
	numTxns := uint64(len(block.Block().Transactions))

	blockSize := uint64(block.Block().SerializeSize())

	mainTip := b.bd.GetMainChainTip()
	mainTipNode := b.GetBlockNode(mainTip)
	if mainTipNode == nil {
		return fmt.Errorf("No main tip node\n")
	}
	state := newBestState(mainTip.GetHash(), mainTipNode.Difficulty(), blockSize, numTxns, b.CalcPastMedianTime(mainTip), lastState.TotalTxns+numTxns,
		b.bd.GetMainChainTip().GetState().GetWeight(), b.bd.GetGraphState(), b.GetTokenTipHash(), *mainTip.GetState().Root())

	// Atomically insert info into the database.
	err := b.db.Update(func(dbTx legacydb.Tx) error {
		// Update best block state.
		err := dbPutBestState(dbTx, state, pow.CalcWork(mainTipNode.Difficulty(), mainTipNode.Pow().GetPowType()))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if b.indexManager != nil {
		err := b.indexManager.UpdateMainTip(mainTip.GetHash(), uint64(mainTip.GetOrder()))
		if err != nil {
			return err
		}
	}
	// Update the state for the best block.  Notice how this replaces the
	// entire struct instead of updating the existing one.  This effectively
	// allows the old version to act as a snapshot which callers can use
	// freely without needing to hold a lock for the duration.  See the
	// comments on the state variable for more details.
	b.stateLock.Lock()
	b.stateSnapshot = state
	b.stateLock.Unlock()

	return b.bd.Commit()
}

func (b *BlockChain) updateBlockState(ib meerdag.IBlock, block *types.SerializedBlock) error {
	if !ib.IsOrdered() {
		return b.updateDefaultBlockState(ib)
	}
	if ib.GetState() == nil ||
		ib.GetOrder() <= 0 {
		return fmt.Errorf("block state is nill:%d %s", ib.GetID(), ib.GetHash().String())
	}
	bs, ok := ib.GetState().(*state.BlockState)
	if !ok {
		return fmt.Errorf("block state is nill:%d %s", ib.GetID(), ib.GetHash().String())
	}
	b.UpdateWeight(ib)
	prev := b.bd.GetBlockByOrder(ib.GetOrder() - 1)
	if prev == nil {
		return fmt.Errorf("No prev block:%d %s", ib.GetID(), ib.GetHash().String())
	}
	bs.Update(block, prev.GetState().(*state.BlockState), b.meerChain.GetCurHeader())
	b.BlockDAG().AddToCommit(ib)
	return nil
}

func (b *BlockChain) updateDefaultBlockState(ib meerdag.IBlock) error {
	if ib.GetState() == nil {
		return fmt.Errorf("block state is nill:%d %s", ib.GetID(), ib.GetHash().String())
	}
	bs, ok := ib.GetState().(*state.BlockState)
	if !ok {
		return fmt.Errorf("block state is nill:%d %s", ib.GetID(), ib.GetHash().String())
	}
	mp := b.bd.GetBlockById(ib.GetMainParent())
	if mp == nil {
		return fmt.Errorf("No main parent:%d %s", ib.GetID(), ib.GetHash().String())
	}
	bs.SetDefault(mp.GetState().(*state.BlockState))
	b.BlockDAG().AddToCommit(ib)
	return nil
}

func (b *BlockChain) prepareEVMEnvironment(block meerdag.IBlock) error {
	prev := b.bd.GetBlockByOrder(block.GetOrder() - 1)
	if prev == nil {
		return fmt.Errorf("No dag block:%s,id:%d\n", block.GetHash().String(), block.GetID())
	}
	_, err := b.meerChain.PrepareEnvironment(prev.GetState())
	if err != nil {
		return err
	}
	return nil
}

func (b *BlockChain) UpdateWeight(ib meerdag.IBlock) {
	if ib.GetID() != meerdag.GenesisId {
		pb := ib.(*meerdag.PhantomBlock)
		tp := b.bd.GetBlockById(pb.GetMainParent())
		pb.GetState().SetWeight(tp.GetState().GetWeight())

		pb.GetState().SetWeight(pb.GetState().GetWeight() + uint64(b.CalcWeight(pb, b.bd.GetBlueInfo(pb))))
		if pb.GetBlueDiffAnticoneSize() > 0 {
			for k := range pb.GetBlueDiffAnticone().GetMap() {
				bdpb := b.bd.GetBlockById(k)
				pb.GetState().SetWeight(pb.GetState().GetWeight() + uint64(b.CalcWeight(bdpb, b.bd.GetBlueInfo(bdpb))))
			}
		}
		b.bd.AddToCommit(ib)
	}
}

type processMsg struct {
	block  *types.SerializedBlock
	flags  BehaviorFlags
	result chan *processResult
}

type processResult struct {
	block    meerdag.IBlock
	isOrphan bool
	err      error
}
