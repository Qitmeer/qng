package chaindb

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var (
	DBDirectoryName = "chaindata"
)

type ChainDB struct {
	db ethdb.Database

	cfg  *config.Config
	lock sync.RWMutex
	// All open databases
	databases   map[*closeTrackingDB]struct{}
	closedState atomic.Bool
}

func (cdb *ChainDB) Name() string {
	return "Chain DB"
}

func (cdb *ChainDB) Init() error {
	return nil
}

func (cdb *ChainDB) Close() {
	log.Info("Close", "name", cdb.Name())
	if cdb.closedState.Load() {
		log.Error("Already closed", "name", cdb.Name())
		return
	}
	cdb.closedState.Store(true)
	cdb.closeDatabases()
}

// wrapDatabase ensures the database will be auto-closed when Node is closed.
func (cdb *ChainDB) wrapDatabase(db ethdb.Database) ethdb.Database {
	wrapper := &closeTrackingDB{db, cdb}
	cdb.databases[wrapper] = struct{}{}
	return wrapper
}

// closeDatabases closes all open databases.
func (cdb *ChainDB) closeDatabases() (errors []error) {
	for db := range cdb.databases {
		delete(cdb.databases, db)
		if err := db.Database.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (cdb *ChainDB) CloseDatabases() (errors []error) {
	return cdb.closeDatabases()
}

func (cdb *ChainDB) OpenDatabaseWithFreezer(name string, cache, handles int, ancient string, namespace string, readonly bool) (ethdb.Database, error) {
	cdb.lock.Lock()
	defer cdb.lock.Unlock()
	if cdb.closedState.Load() {
		return nil, ErrDBClosed
	}

	var db ethdb.Database
	var err error
	if cdb.cfg.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		db, err = rawdb.Open(rawdb.OpenOptions{
			Type:              node.DefaultConfig.DBEngine,
			Directory:         cdb.cfg.ResolveDataPath(name),
			AncientsDirectory: cdb.ResolveAncient(name, ancient),
			Namespace:         namespace,
			Cache:             cache,
			Handles:           handles,
			ReadOnly:          readonly,
		})
	}

	if err == nil {
		db = cdb.wrapDatabase(db)
	}
	return db, err
}

func (cdb *ChainDB) OpenDatabase(name string, cache, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	cdb.lock.Lock()
	defer cdb.lock.Unlock()
	if cdb.closedState.Load() {
		return nil, ErrDBClosed
	}

	var db ethdb.Database
	var err error
	if cdb.cfg.DataDir == "" {
		db = rawdb.NewMemoryDatabase()
	} else {
		db, err = rawdb.Open(rawdb.OpenOptions{
			Type:      node.DefaultConfig.DBEngine,
			Directory: cdb.cfg.ResolveDataPath(name),
			Namespace: namespace,
			Cache:     cache,
			Handles:   handles,
			ReadOnly:  readonly,
		})
	}

	if err == nil {
		db = cdb.wrapDatabase(db)
	}
	return db, err
}

func (cdb *ChainDB) ResolveAncient(name string, ancient string) string {
	switch {
	case ancient == "":
		ancient = filepath.Join(cdb.cfg.ResolveDataPath(name), "ancient")
	case !filepath.IsAbs(ancient):
		ancient = cdb.cfg.ResolveDataPath(ancient)
	}
	return ancient
}

func (cdb *ChainDB) Rebuild(mgr model.IndexManager) error {
	return fmt.Errorf("No support Rebuild:%s", cdb.Name())
}

func (cdb *ChainDB) GetSpendJournal(bh *hash.Hash) ([]byte, error) {
	return rawdb.ReadSpendJournal(cdb.db, bh), nil
}

func (cdb *ChainDB) PutSpendJournal(bh *hash.Hash, data []byte) error {
	return rawdb.WriteSpendJournal(cdb.db, bh, data)
}

func (cdb *ChainDB) DeleteSpendJournal(bh *hash.Hash) error {
	rawdb.DeleteSpendJournal(cdb.db, bh)
	return nil
}

func (cdb *ChainDB) GetUtxo(key []byte) ([]byte, error) {
	return rawdb.ReadUtxo(cdb.db, key), nil
}

func (cdb *ChainDB) PutUtxo(key []byte, data []byte) error {
	return rawdb.WriteUtxo(cdb.db, key, data)
}

func (cdb *ChainDB) DeleteUtxo(key []byte) error {
	rawdb.DeleteUtxo(cdb.db, key)
	return nil
}

func (cdb *ChainDB) GetTokenState(blockID uint) ([]byte, error) {
	return rawdb.ReadTokenState(cdb.db, uint64(blockID)), nil
}

func (cdb *ChainDB) PutTokenState(blockID uint, data []byte) error {
	return rawdb.WriteTokenState(cdb.db, uint64(blockID), data)
}

func (cdb *ChainDB) DeleteTokenState(blockID uint) error {
	rawdb.DeleteTokenState(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) GetBestChainState() ([]byte, error) {
	return rawdb.ReadBestChainState(cdb.db), nil
}

func (cdb *ChainDB) PutBestChainState(data []byte) error {
	return rawdb.WriteBestChainState(cdb.db, data)
}

func (cdb *ChainDB) GetBlock(hash *hash.Hash) (*types.SerializedBlock, error) {
	return rawdb.ReadBody(cdb.db, hash), nil
}

func (cdb *ChainDB) GetBlockBytes(hash *hash.Hash) ([]byte, error) {
	return rawdb.ReadBodyRaw(cdb.db, hash), nil
}

func (cdb *ChainDB) GetHeader(hash *hash.Hash) (*types.BlockHeader, error) {
	return rawdb.ReadHeader(cdb.db, hash), nil
}

func (cdb *ChainDB) PutBlock(block *types.SerializedBlock) error {
	return rawdb.WriteBlock(cdb.db, block)
}

func (cdb *ChainDB) HasBlock(hash *hash.Hash) bool {
	return rawdb.HasHeader(cdb.db, hash)
}

func (cdb *ChainDB) GetDagInfo() ([]byte, error) {
	return rawdb.ReadDAGInfo(cdb.db), nil
}

func (cdb *ChainDB) PutDagInfo(data []byte) error {
	return rawdb.WriteDAGInfo(cdb.db, data)
}

func (cdb *ChainDB) GetDAGBlock(blockID uint) ([]byte, error) {
	return rawdb.ReadDAGBlockBaw(cdb.db, uint64(blockID)), nil
}

func (cdb *ChainDB) PutDAGBlock(blockID uint, data []byte) error {
	return rawdb.WriteDAGBlockRaw(cdb.db, blockID, data)
}

func (cdb *ChainDB) DeleteDAGBlock(blockID uint) error {
	rawdb.DeleteDAGBlock(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) GetDAGBlockIdByHash(bh *hash.Hash) (uint, error) {
	blockID := rawdb.ReadBlockID(cdb.db, bh)
	if blockID == nil {
		return meerdag.MaxId, fmt.Errorf("No blockID:%s", bh.String())
	}
	return uint(*blockID), nil
}

func (cdb *ChainDB) PutDAGBlockIdByHash(bh *hash.Hash, id uint) error {
	rawdb.WriteBlockID(cdb.db, bh, uint64(id))
	return nil
}

func (cdb *ChainDB) DeleteDAGBlockIdByHash(bh *hash.Hash) error {
	rawdb.DeleteBlockID(cdb.db, bh)
	return nil
}

func (cdb *ChainDB) PutMainChainBlock(blockID uint) error {
	return rawdb.WriteMainChain(cdb.db, uint64(blockID))
}

func (cdb *ChainDB) HasMainChainBlock(blockID uint) bool {
	return rawdb.ReadMainChain(cdb.db, uint64(blockID))
}

func (cdb *ChainDB) DeleteMainChainBlock(blockID uint) error {
	rawdb.DeleteMainChain(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) PutBlockIdByOrder(order uint, id uint) error {
	return rawdb.WriteBlockOrderSnapshot(cdb.db, uint64(order), uint64(id))
}

func (cdb *ChainDB) GetBlockIdByOrder(order uint) (uint, error) {
	id := rawdb.ReadBlockOrderSnapshot(cdb.db, uint64(order))
	if id == nil {
		return meerdag.MaxId, nil
	}
	return uint(*id), nil
}

func (cdb *ChainDB) PutDAGTip(id uint, isMain bool) error {
	ctips := rawdb.ReadDAGTips(cdb.db)
	temp := []uint64{}
	for i := 0; i < len(ctips); i++ {
		if ctips[i] == uint64(id) {
			if i == 0 {
				temp = append(temp, uint64(meerdag.MaxId))
			}
			continue
		}
		temp = append(temp, ctips[i])
	}
	tips := []uint64{}
	if len(temp) > 0 {
		tips = append(tips, temp...)
		if isMain {
			tips[0] = uint64(id)
		} else {
			tips = append(tips, uint64(id))
		}
	} else {
		if isMain {
			tips = append(tips, uint64(id))
		} else {
			tips = append(tips, uint64(meerdag.MaxId))
			tips = append(tips, uint64(id))
		}
	}
	return rawdb.WriteDAGTips(cdb.db, tips)
}

func (cdb *ChainDB) GetDAGTips() ([]uint, error) {
	tips := rawdb.ReadDAGTips(cdb.db)
	if len(tips) <= 0 {
		return nil, fmt.Errorf("No tips")
	}
	if tips[0] == uint64(meerdag.MaxId) {
		return nil, fmt.Errorf("Can't find main tip")
	}
	result := []uint{}
	for i := 0; i < len(tips); i++ {
		result = append(result, uint(tips[i]))
	}
	return result, nil
}

func (cdb *ChainDB) DeleteDAGTip(id uint) error {
	tips := rawdb.ReadDAGTips(cdb.db)
	result := []uint64{}
	dirty := false
	for i := 0; i < len(tips); i++ {
		if tips[i] == uint64(id) {
			dirty = true
			if i == 0 {
				result = append(result, uint64(meerdag.MaxId))
			}
			continue
		}
		result = append(result, tips[i])
	}
	if dirty {
		rawdb.WriteDAGTips(cdb.db, result)
	}

	return nil
}

func (cdb *ChainDB) PutDiffAnticone(id uint) error {
	cda := rawdb.ReadDiffAnticone(cdb.db)
	temp := []uint64{}
	for i := 0; i < len(cda); i++ {
		if cda[i] == uint64(id) {
			continue
		}
		temp = append(temp, cda[i])
	}
	da := []uint64{}
	if len(temp) > 0 {
		da = append(da, temp...)
	}
	da = append(da, uint64(id))
	return rawdb.WriteDiffAnticone(cdb.db, da)
}

func (cdb *ChainDB) GetDiffAnticones() ([]uint, error) {
	da := rawdb.ReadDiffAnticone(cdb.db)
	result := []uint{}
	for i := 0; i < len(da); i++ {
		result = append(result, uint(da[i]))
	}
	return result, nil
}

func (cdb *ChainDB) DeleteDiffAnticone(id uint) error {
	da := rawdb.ReadDiffAnticone(cdb.db)
	result := []uint64{}
	dirty := false
	for i := 0; i < len(da); i++ {
		if da[i] == uint64(id) {
			dirty = true
			continue
		}
		result = append(result, da[i])
	}
	if dirty {
		rawdb.WriteDiffAnticone(cdb.db, result)
	}

	return nil
}

func (cdb *ChainDB) Get(key []byte) ([]byte, error) {
	return cdb.db.Get(key)
}

func (cdb *ChainDB) Put(key []byte, value []byte) error {
	return cdb.db.Put(key, value)
}

func New(cfg *config.Config) (*ChainDB, error) {
	cdb := &ChainDB{
		cfg:       cfg,
		databases: make(map[*closeTrackingDB]struct{}),
	}
	cdb.closedState.Store(false)
	return cdb, nil
}
