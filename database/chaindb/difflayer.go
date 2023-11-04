package chaindb

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	com "github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type diffLayer struct {
	db     ethdb.Database
	memory atomic.Uint64 // Approximate guess as to how much memory we use
	root   hash.Hash     // Root hash to which this snapshot diff belongs to
	stale  atomic.Bool   // Signals that the layer became stale (state progressed)
	lock   sync.RWMutex
	cache  uint64

	spendJournal   map[hash.Hash][]byte
	utxo           map[string][]byte
	tokenState     map[uint][]byte
	bestChainState []byte
	blocks         map[hash.Hash]*types.SerializedBlock
	dagBlocks      map[uint][]byte
	blockidByHash  map[hash.Hash]uint
	mainchain      map[uint]bool
	blockidByOrder map[uint]uint
	tips           map[uint]bool
	mainTip        uint
	txIdxEntrys    map[hash.Hash]*diffTxIdxEntry

	wg   sync.WaitGroup
	quit chan struct{}

	cdb *ChainDB
}

func (dl *diffLayer) handler() {
	timer := time.NewTicker(params.ActiveNetParams.TargetTimePerBlock)
out:
	for {
		select {
		case <-timer.C:
			dl.lock.RLock()
			bcSize := len(dl.blocks)
			dl.lock.RUnlock()

			if dl.memory.Load() >= dl.cache ||
				bcSize >= meerdag.MinBlockDataCache {
				err := dl.flatten()
				if err != nil {
					log.Error(err.Error())
				}
			}
		case <-dl.quit:
			break out
		}
	}
	timer.Stop()
	dl.wg.Done()
}

func (dl *diffLayer) close() error {
	log.Info("diff layer close")

	close(dl.quit)
	dl.wg.Wait()
	err := dl.flatten()
	if err != nil {
		return err
	}
	dl.cdb = nil
	return nil
}

func (dl *diffLayer) flatten() error {
	if dl.cdb.shutdownTracker != nil {
		dl.cdb.shutdownTracker.Wait("difflayer")
		defer dl.cdb.shutdownTracker.Done()
	}

	dl.lock.Lock()
	defer dl.lock.Unlock()

	start := time.Now()
	log.Info("Start flatten diff layer", "memory", dl.memorySize().String())

	batch := dl.db.NewBatch()
	if len(dl.spendJournal) > 0 {
		for k, v := range dl.spendJournal {
			if len(v) <= 0 {
				rawdb.DeleteSpendJournal(batch, &k)
			} else {
				err := rawdb.WriteSpendJournal(batch, &k, v)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		for k := range dl.spendJournal {
			delete(dl.spendJournal, k)
		}
	}

	if len(dl.utxo) > 0 {
		for k, v := range dl.utxo {
			if len(v) <= 0 {
				rawdb.DeleteUtxo(batch, []byte(k))
			} else {
				err := rawdb.WriteUtxo(batch, []byte(k), v)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		for k := range dl.utxo {
			delete(dl.utxo, k)
		}
	}

	if len(dl.tokenState) > 0 {
		for k, v := range dl.tokenState {
			if len(v) <= 0 {
				rawdb.DeleteTokenState(batch, uint64(k))
			} else {
				err := rawdb.WriteTokenState(batch, uint64(k), v)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		for k := range dl.tokenState {
			delete(dl.tokenState, k)
		}
	}

	if len(dl.bestChainState) > 0 {
		err := rawdb.WriteBestChainState(batch, dl.bestChainState)
		if err != nil {
			log.Error(err.Error())
		}
	}

	if len(dl.blocks) > 0 {
		for _, v := range dl.blocks {
			err := rawdb.WriteBlock(batch, v)
			if err != nil {
				log.Error(err.Error())
			}
		}
		for k := range dl.blocks {
			delete(dl.blocks, k)
		}
	}

	if len(dl.dagBlocks) > 0 {
		for k, v := range dl.dagBlocks {
			if len(v) <= 0 {
				rawdb.DeleteDAGBlock(batch, uint64(k))
			} else {
				err := rawdb.WriteDAGBlockRaw(batch, k, v)
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		for k := range dl.dagBlocks {
			delete(dl.dagBlocks, k)
		}
	}

	if len(dl.blockidByHash) > 0 {
		for k, v := range dl.blockidByHash {
			if v == meerdag.MaxId {
				rawdb.DeleteBlockID(batch, &k)
			} else {
				rawdb.WriteBlockID(batch, &k, uint64(v))
			}
		}
		for k := range dl.blockidByHash {
			delete(dl.blockidByHash, k)
		}
	}

	if len(dl.mainchain) > 0 {
		for k, v := range dl.mainchain {
			if !v {
				rawdb.DeleteMainChain(batch, uint64(k))
			} else {
				err := rawdb.WriteMainChain(batch, uint64(k))
				if err != nil {
					log.Error(err.Error())
				}
			}
		}
		for k := range dl.mainchain {
			delete(dl.mainchain, k)
		}
	}

	if len(dl.blockidByOrder) > 0 {
		for k, v := range dl.blockidByOrder {
			err := rawdb.WriteBlockOrderSnapshot(batch, uint64(k), uint64(v))
			if err != nil {
				log.Error(err.Error())
			}
		}
		for k := range dl.blockidByOrder {
			delete(dl.blockidByOrder, k)
		}
	}

	if len(dl.tips) > 0 {
		tips, err := dl.doGetDAGTips()
		if err != nil {
			return err
		}
		result := []uint64{}
		for i := 0; i < len(tips); i++ {
			result = append(result, uint64(tips[i]))
		}
		err = rawdb.WriteDAGTips(batch, result)
		if err != nil {
			log.Error(err.Error())
		}
		for k := range dl.tips {
			delete(dl.tips, k)
		}
	}

	if len(dl.txIdxEntrys) > 0 {
		for _, tide := range dl.txIdxEntrys {
			var err error
			if tide.add {
				err = rawdb.WriteTxLookupEntry(batch, tide.tx.Hash(), uint64(tide.blockid))
			} else {
				err = rawdb.DeleteTxLookupEntry(batch, tide.tx.Hash())
			}
			if err != nil {
				log.Error(err.Error())
			}
		}
		for k := range dl.txIdxEntrys {
			delete(dl.txIdxEntrys, k)
		}
	}

	err := batch.Write()
	if err != nil {
		log.Error(err.Error())
	}
	dl.memory.Store(0)
	log.Info("End flatten diff layer", "cost", time.Since(start))
	return nil
}

func (dl *diffLayer) GetSpendJournal(bh *hash.Hash) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.spendJournal) > 0 {
		data, ok := dl.spendJournal[*bh]
		if ok {
			if len(data) > 0 {
				return data, nil
			} else {
				return nil, nil
			}
		}
	}
	return rawdb.ReadSpendJournal(dl.db, bh), nil
}

func (dl *diffLayer) PutSpendJournal(bh *hash.Hash, data []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.spendJournal == nil {
		dl.spendJournal = map[hash.Hash][]byte{}
	}
	dl.spendJournal[*bh] = data

	dl.memory.Add(uint64(len(data) + hash.HashSize))
	return nil
}

func (dl *diffLayer) DeleteSpendJournal(bh *hash.Hash) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.spendJournal == nil {
		dl.spendJournal = map[hash.Hash][]byte{}
	}
	dl.spendJournal[*bh] = nil

	dl.memory.Add(uint64(hash.HashSize))
	return nil
}

func (dl *diffLayer) GetUtxo(key []byte) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.utxo) > 0 {
		data, ok := dl.utxo[string(key)]
		if ok {
			if len(data) > 0 {
				return data, nil
			} else {
				return nil, nil
			}
		}
	}
	return rawdb.ReadUtxo(dl.db, key), nil
}

func (dl *diffLayer) PutUtxo(key []byte, data []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.utxo == nil {
		dl.utxo = map[string][]byte{}
	}
	dl.utxo[string(key)] = data

	dl.memory.Add(uint64(len(data) + len(key)))
	return nil
}

func (dl *diffLayer) DeleteUtxo(key []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.utxo == nil {
		dl.utxo = map[string][]byte{}
	}
	dl.utxo[string(key)] = nil

	dl.memory.Add(uint64(len(key)))
	return nil
}

func (dl *diffLayer) ForeachUtxo(fn func(key []byte, data []byte) error) error {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.utxo) > 0 {
		for k, v := range dl.utxo {
			if len(v) <= 0 {
				continue
			}
			err := fn([]byte(k), v)
			if err != nil {
				return err
			}
		}
	}
	fun := func(key []byte, data []byte) error {
		if len(dl.utxo) > 0 {
			_, ok := dl.utxo[string(key)]
			if ok {
				return nil
			}
		}
		return fn(key, data)
	}
	return rawdb.ForeachUtxo(dl.db, fun)
}

func (dl *diffLayer) UpdateUtxo(opts []*com.UtxoOpt) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.utxo == nil {
		dl.utxo = map[string][]byte{}
	}

	for _, opt := range opts {
		if opt.Add {
			dl.utxo[string(opt.Key)] = opt.Data
			dl.memory.Add(uint64(len(opt.Data) + len(opt.Key)))
		} else {
			dl.utxo[string(opt.Key)] = nil
			dl.memory.Add(uint64(len(opt.Key)))
		}
	}
	return nil
}

func (dl *diffLayer) GetTokenState(blockID uint) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.tokenState) > 0 {
		data, ok := dl.tokenState[blockID]
		if ok {
			if len(data) > 0 {
				return data, nil
			} else {
				return nil, nil
			}
		}
	}
	return rawdb.ReadTokenState(dl.db, uint64(blockID)), nil
}

func (dl *diffLayer) PutTokenState(blockID uint, data []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.tokenState == nil {
		dl.tokenState = map[uint][]byte{}
	}
	dl.tokenState[blockID] = data

	dl.memory.Add(uint64(len(data)) + uint64(unsafe.Sizeof(blockID)))
	return nil
}

func (dl *diffLayer) DeleteTokenState(blockID uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.tokenState == nil {
		dl.tokenState = map[uint][]byte{}
	}
	dl.tokenState[blockID] = nil

	dl.memory.Add(uint64(unsafe.Sizeof(blockID)))
	return nil
}

func (dl *diffLayer) GetBestChainState() ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.bestChainState) > 0 {
		return dl.bestChainState, nil
	}
	return rawdb.ReadBestChainState(dl.db), nil
}

func (dl *diffLayer) PutBestChainState(data []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if len(dl.bestChainState) > 0 {
		cur := int64(dl.memory.Load()) - int64(len(dl.bestChainState))
		dl.memory.Store(uint64(cur))
	}
	dl.bestChainState = data
	dl.memory.Add(uint64(len(data)))
	return nil
}

func (dl *diffLayer) GetBlock(hash *hash.Hash) (*types.SerializedBlock, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.blocks) > 0 {
		data, ok := dl.blocks[*hash]
		if ok {
			return data, nil
		}
	}
	return rawdb.ReadBody(dl.db, hash), nil
}

func (dl *diffLayer) GetBlockBytes(hash *hash.Hash) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.blocks) > 0 {
		data, ok := dl.blocks[*hash]
		if ok {
			return data.Bytes()
		}
	}
	return rawdb.ReadBodyRaw(dl.db, hash), nil
}

func (dl *diffLayer) GetHeader(hash *hash.Hash) (*types.BlockHeader, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.blocks) > 0 {
		data, ok := dl.blocks[*hash]
		if ok {
			return &data.Block().Header, nil
		}
	}
	return rawdb.ReadHeader(dl.db, hash), nil
}

func (dl *diffLayer) PutBlock(block *types.SerializedBlock) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.blocks == nil {
		dl.blocks = map[hash.Hash]*types.SerializedBlock{}
	}
	dl.blocks[*block.Hash()] = block

	bbs, err := block.Bytes()
	if err != nil {
		return err
	}
	dl.memory.Add(uint64(len(bbs)) + hash.HashSize)
	return nil
}

func (dl *diffLayer) HasBlock(hash *hash.Hash) bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.blocks) > 0 {
		_, ok := dl.blocks[*hash]
		if ok {
			return true
		}
	}
	return rawdb.HasHeader(dl.db, hash)
}

func (dl *diffLayer) GetDAGBlock(blockID uint) ([]byte, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.dagBlocks) > 0 {
		data, ok := dl.dagBlocks[blockID]
		if ok {
			if len(data) > 0 {
				return data, nil
			} else {
				return nil, nil
			}
		}
	}
	return rawdb.ReadDAGBlockBaw(dl.db, uint64(blockID)), nil
}

func (dl *diffLayer) PutDAGBlock(blockID uint, data []byte) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.dagBlocks == nil {
		dl.dagBlocks = map[uint][]byte{}
	}
	dl.dagBlocks[blockID] = data

	dl.memory.Add(uint64(len(data)) + uint64(unsafe.Sizeof(blockID)))
	return nil
}

func (dl *diffLayer) DeleteDAGBlock(blockID uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.dagBlocks == nil {
		dl.dagBlocks = map[uint][]byte{}
	}
	dl.dagBlocks[blockID] = nil

	dl.memory.Add(uint64(unsafe.Sizeof(blockID)))
	return nil
}

func (dl *diffLayer) GetDAGBlockIdByHash(bh *hash.Hash) (uint, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.dagBlocks) > 0 {
		data, ok := dl.blockidByHash[*bh]
		if ok {
			if data != meerdag.MaxId {
				return data, nil
			} else {
				return meerdag.MaxId, nil
			}
		}
	}

	blockID := rawdb.ReadBlockID(dl.db, bh)
	if blockID == nil {
		return meerdag.MaxId, fmt.Errorf("No blockID:%s", bh.String())
	}
	return uint(*blockID), nil
}

func (dl *diffLayer) PutDAGBlockIdByHash(bh *hash.Hash, id uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.blockidByHash == nil {
		dl.blockidByHash = map[hash.Hash]uint{}
	}
	dl.blockidByHash[*bh] = id

	dl.memory.Add(uint64(unsafe.Sizeof(id)) + hash.HashSize)
	return nil
}

func (dl *diffLayer) DeleteDAGBlockIdByHash(bh *hash.Hash) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.blockidByHash == nil {
		dl.blockidByHash = map[hash.Hash]uint{}
	}
	dl.blockidByHash[*bh] = meerdag.MaxId

	dl.memory.Add(uint64(unsafe.Sizeof(meerdag.MaxId)) + hash.HashSize)
	return nil
}

func (dl *diffLayer) PutMainChainBlock(blockID uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.mainchain == nil {
		dl.mainchain = map[uint]bool{}
	}
	dl.mainchain[blockID] = true

	dl.memory.Add(uint64(unsafe.Sizeof(blockID)) + 1)
	return nil
}

func (dl *diffLayer) HasMainChainBlock(blockID uint) bool {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.mainchain) > 0 {
		data, ok := dl.mainchain[blockID]
		if ok {
			return data
		}
	}
	return rawdb.ReadMainChain(dl.db, uint64(blockID))
}

func (dl *diffLayer) DeleteMainChainBlock(blockID uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.mainchain == nil {
		dl.mainchain = map[uint]bool{}
	}
	dl.mainchain[blockID] = false

	dl.memory.Add(uint64(unsafe.Sizeof(blockID)) + 1)
	return nil
}

func (dl *diffLayer) PutBlockIdByOrder(order uint, id uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.blockidByOrder == nil {
		dl.blockidByOrder = map[uint]uint{}
	}
	dl.blockidByOrder[order] = id

	dl.memory.Add(uint64(unsafe.Sizeof(order)) * 2)
	return nil
}

func (dl *diffLayer) GetBlockIdByOrder(order uint) (uint, error) {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	if len(dl.blockidByOrder) > 0 {
		data, ok := dl.blockidByOrder[order]
		if ok {
			return data, nil
		}
	}

	id := rawdb.ReadBlockOrderSnapshot(dl.db, uint64(order))
	if id == nil {
		return meerdag.MaxId, nil
	}
	return uint(*id), nil
}

func (dl *diffLayer) PutDAGTip(id uint, isMain bool) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	num := 0
	if dl.tips == nil {
		dl.tips = map[uint]bool{}
	} else {
		num = len(dl.tips)
	}
	dl.tips[id] = true
	if isMain {
		dl.mainTip = id
	}

	dl.memory.Add(uint64(5 * (len(dl.tips) - num)))
	return nil
}

func (dl *diffLayer) GetDAGTips() ([]uint, error) {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	return dl.doGetDAGTips()
}

func (dl *diffLayer) doGetDAGTips() ([]uint, error) {
	tips := rawdb.ReadDAGTips(dl.db)
	result := []uint{}
	resultSet := meerdag.NewIdSet()
	for i := 0; i < len(tips); i++ {
		resultSet.Add(uint(tips[i]))
	}

	result = append(result, dl.mainTip)
	if len(dl.tips) > 0 {
		for k, v := range dl.tips {
			if k == dl.mainTip {
				resultSet.Remove(k)
				continue
			}
			if v {
				resultSet.Add(k)
			} else {
				resultSet.Remove(k)
			}
		}
	}

	result = append(result, resultSet.List()...)

	if len(result) <= 0 {
		return nil, fmt.Errorf("No tips")
	}
	if result[0] == meerdag.MaxId {
		return nil, fmt.Errorf("Can't find main tip")
	}

	return result, nil
}

func (dl *diffLayer) DeleteDAGTip(id uint) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	num := 0
	if dl.tips == nil {
		dl.tips = map[uint]bool{}
	} else {
		num = len(dl.tips)
	}
	dl.tips[id] = false
	if dl.mainTip == id {
		dl.mainTip = meerdag.MaxId
	}

	dl.memory.Add(uint64(5 * (len(dl.tips) - num)))
	return nil
}

type diffTxIdxEntry struct {
	tx        *types.Tx
	blockid   uint
	blockhash *hash.Hash
	add       bool
}

func (dl *diffLayer) PutTxIdxEntrys(sblock *types.SerializedBlock, block model.Block) error {
	dl.lock.Lock()
	defer dl.lock.Unlock()

	if dl.txIdxEntrys == nil {
		dl.txIdxEntrys = map[hash.Hash]*diffTxIdxEntry{}
	}
	for _, tx := range sblock.Transactions() {
		if tx.IsDuplicate {
			continue
		}
		dtie := &diffTxIdxEntry{
			tx:        tx,
			blockid:   block.GetID(),
			blockhash: sblock.Hash(),
			add:       true,
		}
		dl.txIdxEntrys[*tx.Hash()] = dtie
		dl.memory.Add(uint64(13))
	}
	return nil
}

func (dl *diffLayer) GetTxIdxEntry(id *hash.Hash, verbose bool) (*types.Tx, *hash.Hash, error) {
	dl.lock.Lock()

	if len(dl.txIdxEntrys) > 0 {
		dtie, ok := dl.txIdxEntrys[*id]
		if ok {
			if dtie.add {
				dl.lock.Unlock()
				return dtie.tx, dtie.blockhash, nil
			} else {
				dl.lock.Unlock()
				return nil, nil, nil
			}
		}
	}

	dl.lock.Unlock()

	if !verbose {
		blockID := rawdb.ReadTxLookupEntry(dl.db, id)
		if blockID == nil {
			return nil, nil, nil
		}
		blockhash, err := meerdag.DBGetDAGBlockHashByID(dl.cdb, *blockID)
		if err != nil {
			return nil, nil, err
		}
		return nil, blockhash, nil
	}
	tx, _, blockhash, _ := rawdb.ReadTransaction(dl.db, id)
	return tx, blockhash, nil
}

func (dl *diffLayer) DeleteTxIdxEntrys(block *types.SerializedBlock) error {
	for _, tx := range block.Transactions() {
		_, blockHash, _ := dl.GetTxIdxEntry(tx.Hash(), false)
		if blockHash != nil && !blockHash.IsEqual(block.Hash()) {
			continue
		}

		dl.lock.Lock()
		if dl.txIdxEntrys == nil {
			dl.txIdxEntrys = map[hash.Hash]*diffTxIdxEntry{}
		}
		dtie := &diffTxIdxEntry{
			tx:        tx,
			blockid:   meerdag.MaxId,
			blockhash: block.Hash(),
			add:       false,
		}
		dl.txIdxEntrys[*tx.Hash()] = dtie
		dl.memory.Add(uint64(13))
		dl.lock.Unlock()
	}
	return nil
}

func (dl *diffLayer) memorySize() common.StorageSize {
	return common.StorageSize(dl.memory.Load())
}

func (dl *diffLayer) objects() int {
	dl.lock.RLock()
	defer dl.lock.RUnlock()

	return len(dl.spendJournal) + len(dl.utxo) + len(dl.tokenState) + len(dl.blocks) + len(dl.dagBlocks) +
		len(dl.blockidByHash) + len(dl.mainchain) + len(dl.blockidByOrder) + len(dl.tips) + len(dl.txIdxEntrys)
}

func newDiffLayer(cdb *ChainDB, cache int) *diffLayer {
	dl := &diffLayer{
		cdb:   cdb,
		db:    cdb.db,
		root:  hash.ZeroHash,
		cache: uint64(cache) * 1024 * 1024,
		quit:  make(chan struct{}),
	}
	dl.stale.Store(false)
	dl.memory.Store(0)
	log.Info("New diff layer", "cache", common.StorageSize(dl.cache).String())
	dl.wg.Add(1)
	go dl.handler()
	return dl
}
