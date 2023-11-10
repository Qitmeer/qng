package chaindb

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/shutdown"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/common"
	"github.com/Qitmeer/qng/database/rawdb"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/ethdb"
	"path/filepath"
	"sync"
	"sync/atomic"
)

var (
	DBDirectoryName = "meerchain"
	CreateIfNoExist = true
)

type ChainDB struct {
	db ethdb.Database

	cfg  *config.Config
	lock sync.RWMutex
	// All open databases
	databases   map[*closeTrackingDB]struct{}
	closedState atomic.Bool

	hasInit bool

	diff            *diffLayer
	shutdownTracker *shutdown.Tracker
	ancient         bool
}

func (cdb *ChainDB) Name() string {
	return "Chain DB"
}

func (cdb *ChainDB) Init() error {
	log.Info("Init", "name", cdb.Name())
	if cdb.hasInit {
		return fmt.Errorf("%s: Need to thoroughly clean up old data", cdb.Name())
	}
	return nil
}

func (cdb *ChainDB) Close() {
	log.Info("Close", "name", cdb.Name())
	if cdb.diff != nil {
		err := cdb.diff.close()
		if err != nil {
			log.Error(err.Error())
		}
	}
	if cdb.closedState.Load() {
		log.Error("Already closed", "name", cdb.Name())
		return
	}
	cdb.closedState.Store(true)
	cdb.closeDatabases()
}

func (cdb *ChainDB) DB() ethdb.Database {
	return cdb.db
}

func (cdb *ChainDB) DBEngine() string {
	if cdb.cfg.DbType == "ffldb" {
		return "leveldb"
	}
	return cdb.cfg.DbType
}

func (cdb *ChainDB) Snapshot() error {
	if cdb.cfg.SnapshotCache() > 0 {
		cdb.diff = newDiffLayer(cdb, cdb.cfg.SnapshotCache())
	}
	return nil
}

func (cdb *ChainDB) SnapshotInfo() string {
	if cdb.diff == nil {
		return "not active"
	}
	return fmt.Sprintf("mem=%s,objects=%d", cdb.diff.memorySize().String(), cdb.diff.objects())
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
	if !CreateIfNoExist {
		existingDb := rawdb.PreexistingDatabase(cdb.cfg.ResolveDataPath(name))
		if len(existingDb) <= 0 {
			return nil, ErrDBAbsent
		}
	}

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
			Type:              cdb.DBEngine(),
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
			Type:      cdb.DBEngine(),
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
	if !cdb.ancient {
		return ""
	}
	switch {
	case ancient == "":
		ancient = filepath.Join(cdb.cfg.ResolveDataPath(name), "ancient")
	case !filepath.IsAbs(ancient):
		ancient = cdb.cfg.ResolveDataPath(ancient)
	}
	return ancient
}

func (cdb *ChainDB) Rebuild(mgr model.IndexManager) error {
	err := cdb.CleanInvalidTxIdx()
	if err != nil {
		return err
	}
	err = cdb.CleanAddrIdx(false)
	if err != nil {
		return err
	}

	err = rawdb.CleanSpendJournal(cdb.db)
	if err != nil {
		return err
	}
	err = rawdb.CleanUtxo(cdb.db)
	if err != nil {
		return err
	}
	return rawdb.CleanTokenState(cdb.db)
}

func (cdb *ChainDB) GetSpendJournal(bh *hash.Hash) ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetSpendJournal(bh)
	}
	return rawdb.ReadSpendJournal(cdb.db, bh), nil
}

func (cdb *ChainDB) PutSpendJournal(bh *hash.Hash, data []byte) error {
	if cdb.diff != nil {
		return cdb.diff.PutSpendJournal(bh, data)
	}
	return rawdb.WriteSpendJournal(cdb.db, bh, data)
}

func (cdb *ChainDB) DeleteSpendJournal(bh *hash.Hash) error {
	if cdb.diff != nil {
		cdb.diff.DeleteSpendJournal(bh)
		return nil
	}
	rawdb.DeleteSpendJournal(cdb.db, bh)
	return nil
}

func (cdb *ChainDB) GetUtxo(key []byte) ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetUtxo(key)
	}
	return rawdb.ReadUtxo(cdb.db, key), nil
}

func (cdb *ChainDB) PutUtxo(key []byte, data []byte) error {
	if cdb.diff != nil {
		return cdb.diff.PutUtxo(key, data)
	}
	return rawdb.WriteUtxo(cdb.db, key, data)
}

func (cdb *ChainDB) DeleteUtxo(key []byte) error {
	if cdb.diff != nil {
		cdb.diff.DeleteUtxo(key)
		return nil
	}
	rawdb.DeleteUtxo(cdb.db, key)
	return nil
}

func (cdb *ChainDB) ForeachUtxo(fn func(key []byte, data []byte) error) error {
	if cdb.diff != nil {
		return cdb.diff.ForeachUtxo(fn)
	}
	return rawdb.ForeachUtxo(cdb.db, fn)
}

func (cdb *ChainDB) UpdateUtxo(opts []*common.UtxoOpt) error {
	if len(opts) <= 0 {
		return nil
	}
	if cdb.diff != nil {
		return cdb.diff.UpdateUtxo(opts)
	}
	batch := cdb.db.NewBatch()
	for _, opt := range opts {
		if opt.Add {
			err := rawdb.WriteUtxo(batch, opt.Key, opt.Data)
			if err != nil {
				return err
			}
		} else {
			rawdb.DeleteUtxo(batch, opt.Key)
		}
	}
	return batch.Write()
}

func (cdb *ChainDB) GetTokenState(blockID uint) ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetTokenState(blockID)
	}
	return rawdb.ReadTokenState(cdb.db, uint64(blockID)), nil
}

func (cdb *ChainDB) PutTokenState(blockID uint, data []byte) error {
	if cdb.diff != nil {
		return cdb.diff.PutTokenState(blockID, data)
	}
	return rawdb.WriteTokenState(cdb.db, uint64(blockID), data)
}

func (cdb *ChainDB) DeleteTokenState(blockID uint) error {
	if cdb.diff != nil {
		return cdb.diff.DeleteTokenState(blockID)
	}
	rawdb.DeleteTokenState(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) GetBestChainState() ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetBestChainState()
	}
	return rawdb.ReadBestChainState(cdb.db), nil
}

func (cdb *ChainDB) PutBestChainState(data []byte) error {
	if cdb.diff != nil {
		return cdb.diff.PutBestChainState(data)
	}
	return rawdb.WriteBestChainState(cdb.db, data)
}

func (cdb *ChainDB) GetBlock(hash *hash.Hash) (*types.SerializedBlock, error) {
	if cdb.diff != nil {
		return cdb.diff.GetBlock(hash)
	}
	return rawdb.ReadBody(cdb.db, hash), nil
}

func (cdb *ChainDB) GetBlockBytes(hash *hash.Hash) ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetBlockBytes(hash)
	}
	return rawdb.ReadBodyRaw(cdb.db, hash), nil
}

func (cdb *ChainDB) GetHeader(hash *hash.Hash) (*types.BlockHeader, error) {
	if cdb.diff != nil {
		return cdb.diff.GetHeader(hash)
	}
	return rawdb.ReadHeader(cdb.db, hash), nil
}

func (cdb *ChainDB) PutBlock(block *types.SerializedBlock) error {
	if cdb.diff != nil {
		return cdb.diff.PutBlock(block)
	}
	return cdb.writeBlockToBatch(block)
}

func (cdb *ChainDB) writeBlockToBatch(block *types.SerializedBlock) error {
	batch := cdb.db.NewBatch()
	err := rawdb.WriteBlock(batch, block)
	if err != nil {
		return err
	}
	return batch.Write()
}

func (cdb *ChainDB) HasBlock(hash *hash.Hash) bool {
	if cdb.diff != nil {
		return cdb.diff.HasBlock(hash)
	}
	return rawdb.HasHeader(cdb.db, hash)
}

func (cdb *ChainDB) GetDagInfo() ([]byte, error) {
	return rawdb.ReadDAGInfo(cdb.db), nil
}

func (cdb *ChainDB) PutDagInfo(data []byte) error {
	return rawdb.WriteDAGInfo(cdb.db, data)
}

func (cdb *ChainDB) GetDAGBlock(blockID uint) ([]byte, error) {
	if cdb.diff != nil {
		return cdb.diff.GetDAGBlock(blockID)
	}
	return rawdb.ReadDAGBlockBaw(cdb.db, uint64(blockID)), nil
}

func (cdb *ChainDB) PutDAGBlock(blockID uint, data []byte) error {
	if cdb.diff != nil {
		return cdb.diff.PutDAGBlock(blockID, data)
	}
	return rawdb.WriteDAGBlockRaw(cdb.db, blockID, data)
}

func (cdb *ChainDB) DeleteDAGBlock(blockID uint) error {
	if cdb.diff != nil {
		return cdb.diff.DeleteDAGBlock(blockID)
	}
	rawdb.DeleteDAGBlock(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) GetDAGBlockIdByHash(bh *hash.Hash) (uint, error) {
	if cdb.diff != nil {
		return cdb.diff.GetDAGBlockIdByHash(bh)
	}
	blockID := rawdb.ReadBlockID(cdb.db, bh)
	if blockID == nil {
		return meerdag.MaxId, fmt.Errorf("No blockID:%s", bh.String())
	}
	return uint(*blockID), nil
}

func (cdb *ChainDB) PutDAGBlockIdByHash(bh *hash.Hash, id uint) error {
	if cdb.diff != nil {
		return cdb.diff.PutDAGBlockIdByHash(bh, id)
	}
	rawdb.WriteBlockID(cdb.db, bh, uint64(id))
	return nil
}

func (cdb *ChainDB) DeleteDAGBlockIdByHash(bh *hash.Hash) error {
	if cdb.diff != nil {
		return cdb.diff.DeleteDAGBlockIdByHash(bh)
	}
	rawdb.DeleteBlockID(cdb.db, bh)
	return nil
}

func (cdb *ChainDB) PutMainChainBlock(blockID uint) error {
	if cdb.diff != nil {
		return cdb.diff.PutMainChainBlock(blockID)
	}
	return rawdb.WriteMainChain(cdb.db, uint64(blockID))
}

func (cdb *ChainDB) HasMainChainBlock(blockID uint) bool {
	if cdb.diff != nil {
		return cdb.diff.HasMainChainBlock(blockID)
	}
	return rawdb.ReadMainChain(cdb.db, uint64(blockID))
}

func (cdb *ChainDB) DeleteMainChainBlock(blockID uint) error {
	if cdb.diff != nil {
		return cdb.diff.DeleteMainChainBlock(blockID)
	}
	rawdb.DeleteMainChain(cdb.db, uint64(blockID))
	return nil
}

func (cdb *ChainDB) PutBlockIdByOrder(order uint, id uint) error {
	if cdb.diff != nil {
		return cdb.diff.PutBlockIdByOrder(order, id)
	}
	return rawdb.WriteBlockOrderSnapshot(cdb.db, uint64(order), uint64(id))
}

func (cdb *ChainDB) GetBlockIdByOrder(order uint) (uint, error) {
	if cdb.diff != nil {
		return cdb.diff.GetBlockIdByOrder(order)
	}
	id := rawdb.ReadBlockOrderSnapshot(cdb.db, uint64(order))
	if id == nil {
		return meerdag.MaxId, nil
	}
	return uint(*id), nil
}

func (cdb *ChainDB) PutDAGTip(id uint, isMain bool) error {
	if cdb.diff != nil {
		return cdb.diff.PutDAGTip(id, isMain)
	}
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
	if cdb.diff != nil {
		return cdb.diff.GetDAGTips()
	}
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
	if cdb.diff != nil {
		return cdb.diff.DeleteDAGTip(id)
	}
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

func (cdb *ChainDB) IsLegacy() bool {
	return false
}

func (cdb *ChainDB) GetEstimateFee() ([]byte, error) {
	return rawdb.ReadEstimateFee(cdb.db), nil
}

func (cdb *ChainDB) PutEstimateFee(data []byte) error {
	return rawdb.WriteEstimateFee(cdb.db, data)
}

func (cdb *ChainDB) DeleteEstimateFee() error {
	return rawdb.DeleteEstimateFee(cdb.db)
}

func (cdb *ChainDB) TryUpgrade(di *common.DatabaseInfo, interrupt <-chan struct{}) error {
	return nil
}

func (cdb *ChainDB) StartTrack(info string) error {
	if cdb.diff != nil {
		return nil
	}
	if cdb.shutdownTracker != nil {
		return cdb.shutdownTracker.Wait(info)
	}
	return nil
}

func (cdb *ChainDB) StopTrack() error {
	if cdb.diff != nil {
		return nil
	}
	if cdb.shutdownTracker != nil {
		return cdb.shutdownTracker.Done()
	}
	return nil
}

func New(cfg *config.Config) (*ChainDB, error) {
	cdb := &ChainDB{
		cfg:       cfg,
		databases: make(map[*closeTrackingDB]struct{}),
		hasInit:   meer.Exist(cfg),
		ancient:   true,
	}
	if len(cfg.DataDir) > 0 {
		cdb.shutdownTracker = shutdown.NewTracker(cfg.DataDir)
	}
	cdb.closedState.Store(false)

	var err error
	cdb.db, err = cdb.OpenDatabaseWithFreezer(DBDirectoryName, cfg.DatabaseCache(),
		utils.MakeDatabaseHandles(0), "", "qng/", false)
	if err != nil {
		return nil, err
	}
	if cdb.shutdownTracker != nil {
		err = cdb.shutdownTracker.Check()
		if err != nil {
			return nil, err
		}
	}
	return cdb, nil
}

func NewNaked(cfg *config.Config) (*ChainDB, error) {
	cdb := &ChainDB{
		cfg:       cfg,
		databases: make(map[*closeTrackingDB]struct{}),
		hasInit:   false,
	}
	cdb.closedState.Store(false)

	var err error
	cdb.db, err = cdb.OpenDatabaseWithFreezer(DBDirectoryName, cfg.DatabaseCache(),
		utils.MakeDatabaseHandles(0), "", "qng/", false)
	if err != nil {
		return nil, err
	}
	return cdb, nil
}
