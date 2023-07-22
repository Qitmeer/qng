package meerdag

import (
	"container/list"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/consensus/model"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag/anticone"
	"github.com/Qitmeer/qng/node/service"
	"github.com/Qitmeer/qng/params"
	"io"
	"math"
	"sync"
	"time"
)

// Some available DAG algorithm types
const (
	// A Scalable BlockDAG protocol
	PHANTOM = "phantom"

	// The order of all transactions is solely determined by the Tree Graph (TG)
	CONFLUX = "conflux"

	// Confirming Transactions via Recursive Elections
	SPECTRE = "spectre"

	// GHOSTDAG is an greedy algorithm implementation based on PHANTOM protocol
	GHOSTDAG = "ghostdag"
)

// Maximum number of the DAG tip
const MaxTips = 100

// Maximum order of the DAG block
const MaxBlockOrder = uint(^uint32(0))

// Maximum id of the DAG block
const MaxId = uint(math.MaxUint32)

// Genesis id of the DAG block
const GenesisId = uint(0)

// MaxTipLayerGap
const MaxTipLayerGap = 10

// StableConfirmations
const StableConfirmations = 10

// Max Priority
const MaxPriority = int(math.MaxInt32)

// block data
const MinBlockDataCache = 2000

const MinBlockPruneSize = 2000

// It will create different BlockDAG instances
func NewBlockDAG(dagType string) ConsensusAlgorithm {
	switch dagType {
	case PHANTOM:
		return &Phantom{}
	case CONFLUX:
		return &Conflux{}
	case SPECTRE:
		return &Spectre{}
	case GHOSTDAG:
		return &GhostDAG{}
	}
	return nil
}

func GetDAGTypeIndex(dagType string) byte {
	switch dagType {
	case PHANTOM:
		return 0
	case CONFLUX:
		return 2
	case SPECTRE:
		return 3
	case GHOSTDAG:
		return 4
	}
	return 0
}

func GetDAGTypeByIndex(dagType byte) string {
	switch dagType {
	case 0:
		return PHANTOM
	case 2:
		return CONFLUX
	case 3:
		return SPECTRE
	case 4:
		return GHOSTDAG
	}
	return PHANTOM
}

// The abstract inferface is used to build and manager DAG consensus algorithm
type ConsensusAlgorithm interface {
	// Return the name
	GetName() string

	// This instance is initialized and will be executed first.
	Init(bd *MeerDAG) bool

	// Add a block
	AddBlock(ib IBlock) (*list.List, *list.List)

	// Build self block
	CreateBlock(b *Block) IBlock

	// If the successor return nil, the underlying layer will use the default tips list.
	GetTipsList() []IBlock

	// Query whether a given block is on the main chain.
	isOnMainChain(id uint) bool

	// return the tip of main chain
	GetMainChainTip() IBlock

	// return the tip of main chain id
	GetMainChainTipId() uint

	// return the main parent in the parents
	GetMainParent(parents *IdSet) IBlock

	// encode
	Encode(w io.Writer) error

	// decode
	Decode(r io.Reader) error

	// load
	Load() error

	// IsDAG
	IsDAG(parents []IBlock) bool

	// The main parent concurrency of block
	GetMainParentConcurrency(b IBlock) int

	// GetBlues
	GetBlues(parents *IdSet) uint

	// IsBlue
	IsBlue(id uint) bool

	// getMaxParents
	getMaxParents() int
}

type GetBlockData func(*hash.Hash) IBlockData

type CreateBlockState func(id uint64) model.BlockState

var createBlockState CreateBlockState

type CreateBlockStateFromBytes func(data []byte) (model.BlockState, error)

var createBlockStateFromBytes CreateBlockStateFromBytes

// The general foundation framework of Block DAG implement
type MeerDAG struct {
	service.Service

	// The genesis of block dag
	genesis hash.Hash

	// Use block hash to save all blocks with mapping
	blocks map[uint]IBlock

	// The total number blocks that this dag currently owned
	blockTotal uint

	// The terminal block is in block dag,this block have not any connecting at present.
	tips *IdSet

	tipsDisLimit int64

	// This is time when the last block have added
	lastTime time.Time

	// All orders relate to new block will be committed that need to be consensus
	commitOrder map[uint]uint

	commitBlock *IdSet

	// Current dag instance used. Different algorithms work according to
	// different dag types config.
	instance ConsensusAlgorithm

	// state lock
	stateLock sync.RWMutex

	getBlockData GetBlockData

	// blocks per second
	blockRate float64

	db model.DataBase

	// Rollback mechanism
	lastSnapshot *DAGSnapshot

	// block data
	blockDataLock  sync.RWMutex
	blockDataCache map[uint]time.Time

	wg   sync.WaitGroup
	quit chan struct{}

	minCacheSize   uint64
	minBDCacheSize uint64
}

// Acquire the name of DAG instance
func (bd *MeerDAG) GetName() string {
	return bd.instance.GetName()
}

// GetInstance
func (bd *MeerDAG) GetInstance() ConsensusAlgorithm {
	return bd.instance
}

// Initialize self, the function to be invoked at the beginning
func (bd *MeerDAG) init(dagType string, blockRate float64, db model.DataBase, getBlockData GetBlockData) ConsensusAlgorithm {
	bd.lastTime = time.Unix(roughtime.Now().Unix(), 0)
	bd.commitOrder = map[uint]uint{}
	bd.getBlockData = getBlockData
	bd.db = db
	bd.commitBlock = NewIdSet()
	bd.lastSnapshot = NewDAGSnapshot()
	bd.blockRate = blockRate
	bd.tipsDisLimit = StableConfirmations
	if bd.blockRate < 0 {
		bd.blockRate = anticone.DefaultBlockRate
	}
	bd.instance = NewBlockDAG(dagType)
	bd.instance.Init(bd)

	serializedData, err := bd.db.GetDagInfo()
	if err != nil && len(serializedData) <= 0 {
		err = DBPutDAGInfo(bd)
		if err != nil {
			log.Error(err.Error())
		}
	}
	return bd.instance
}

func (bd *MeerDAG) Start() error {
	if err := bd.Service.Start(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Start MeerDAG:%s", bd.GetName()))
	bd.wg.Add(1)
	go bd.handler()

	return nil
}

func (bd *MeerDAG) Stop() error {
	if err := bd.Service.Stop(); err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Stop MeerDAG:%s", bd.GetName()))
	close(bd.quit)
	bd.wg.Wait()
	return nil
}

// This is an entry for update the block dag,you need pass in a block parameter,
// If add block have failure,it will return false.
func (bd *MeerDAG) AddBlock(b IBlockData) (*list.List, *list.List, IBlock, bool) {
	if onEnd := l.LogAndMeasureExecutionTime(log, "MeerDAG.AddBlock"); onEnd != nil {
		defer onEnd()
	}
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	if b == nil {
		log.Error("block data is nil")
		return nil, nil, nil, false
	}
	// Must keep no block in outside.
	if bd.hasBlock(b.GetHash()) {
		log.Error(fmt.Sprintf("Already own this block:%s", b.GetHash()))
		return nil, nil, nil, false
	}
	parents := []IBlock{}
	var mp IBlock
	if bd.blockTotal > 0 {
		parentsIds := b.GetParents()
		if len(parentsIds) == 0 {
			log.Error(fmt.Sprintf("No paretns:%s", b.GetHash()))
			return nil, nil, nil, false
		}
		ids := NewIdSet()
		for _, v := range parentsIds {
			pib := bd.getBlock(v)
			if pib == nil {
				log.Error(fmt.Sprintf("No parent:%s about parent(%s)", b.GetHash(), v.String()))
				return nil, nil, nil, false
			}
			parents = append(parents, pib)
			ids.Add(pib.GetID())
		}

		if !bd.isDAG(parents, b) {
			log.Error(fmt.Sprintf("Not DAG block:%s", b.GetHash()))
			return nil, nil, nil, false
		}
		// main parent
		mp = bd.instance.GetMainParent(ids)
		if mp == nil {
			log.Error("No main parent", "hash", b.GetHash().String())
			return nil, nil, nil, false
		}
		if !mp.GetHash().IsEqual(b.GetMainParent()) {
			log.Error("Main parent is inconsistent in block", "mainParentInDAG", mp.GetHash().String(), "mainParentInBlock", b.GetMainParent().String(), "block", b.GetHash().String())
			return nil, nil, nil, false
		}
	}
	lastMT := bd.instance.GetMainChainTipId()
	//
	block := Block{id: bd.blockTotal,
		hash:       *b.GetHash(),
		layer:      0,
		mainParent: MaxId,
		data:       b,
		state:      createBlockState(uint64(bd.blockTotal))}
	if mp != nil {
		block.mainParent = mp.GetID()
	}
	if bd.blocks == nil {
		bd.blocks = map[uint]IBlock{}
	}
	ib := bd.instance.CreateBlock(&block)
	bd.blocks[block.id] = ib
	bd.blockDataLock.Lock()
	bd.blockDataCache[block.GetID()] = time.Now()
	bd.blockDataLock.Unlock()
	// db
	bd.commitBlock.AddPair(ib.GetID(), ib)

	//
	if bd.blockTotal == 0 {
		bd.genesis = *block.GetHash()
	}
	bd.lastSnapshot.Clean()
	bd.lastSnapshot.block = ib
	bd.lastSnapshot.tips = bd.tips.Clone()
	bd.lastSnapshot.lastTime = bd.lastTime
	//
	bd.blockTotal++

	if len(parents) > 0 {
		var maxLayer uint = 0
		for _, v := range parents {
			parent := v.(IBlock)
			ib.AddParent(parent)
			parent.AddChild(ib)
			bd.commitBlock.AddPair(parent.GetID(), parent)
			if maxLayer == 0 || maxLayer < parent.GetLayer() {
				maxLayer = parent.GetLayer()
			}
		}
		block.SetLayer(maxLayer + 1)
	}

	//
	bd.updateTips(ib)
	//
	t := time.Unix(b.GetTimestamp(), 0)
	if bd.lastTime.Before(t) {
		bd.lastTime = t
	}
	//
	news, olds := bd.instance.AddBlock(ib)
	bd.optimizeReorganizeResult(news, olds)
	if news == nil {
		news = list.New()
	}
	if olds == nil {
		olds = list.New()
	}
	curMT := bd.getMainChainTip()

	mainOrderGauge.Update(int64(curMT.GetOrder()))
	mainHeightGauge.Update(int64(curMT.GetHeight()))
	mainLayerGauge.Update(int64(curMT.GetLayer()))
	tipsTotalGauge.Update(int64(bd.tips.Size()))

	return news, olds, ib, lastMT != curMT.GetID()
}

// Acquire the genesis block of chain
func (bd *MeerDAG) getGenesis() IBlock {
	return bd.getBlockById(GenesisId)
}

// Acquire the genesis block hash of chain
func (bd *MeerDAG) GetGenesisHash() *hash.Hash {
	return &bd.genesis
}

// If the block is illegal dag,will return false.
// Exclude genesis block
func (bd *MeerDAG) isDAG(parents []IBlock, b IBlockData) bool {
	if bd.GetInstance().GetName() == GHOSTDAG {
		return bd.instance.IsDAG(parents)
	}
	return bd.checkPriority(parents, b) &&
		bd.checkLayerGap(parents) &&
		bd.checkLegality(parents) &&
		bd.instance.IsDAG(parents)
}

// Total number of blocks
func (bd *MeerDAG) GetBlockTotal() uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.blockTotal
}

// The last time is when add one block to DAG.
func (bd *MeerDAG) GetLastTime() *time.Time {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return &bd.lastTime
}

func (bd *MeerDAG) GetFutureSet(fs *IdSet, b IBlock) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	bd.getFutureSet(fs, b)
}

// Returns a future collection of block. This function is a recursively called function
// So we should consider its efficiency.
func (bd *MeerDAG) getFutureSet(fs *IdSet, b IBlock) {
	children := bd.getChildren(b)
	if children == nil || children.IsEmpty() {
		return
	}
	for k, v := range children.GetMap() {
		ib := v.(IBlock)
		if !fs.Has(k) {
			fs.AddPair(k, ib)
			bd.getFutureSet(fs, ib)
		}
	}
}

// Query whether a given block is on the main chain.
// Note that some DAG protocols may not support this feature.
func (bd *MeerDAG) IsOnMainChain(id uint) bool {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.isOnMainChain(id)
}

// Query whether a given block is on the main chain.
// Note that some DAG protocols may not support this feature.
func (bd *MeerDAG) isOnMainChain(id uint) bool {
	return bd.instance.isOnMainChain(id)
}

// return the tip of main chain
func (bd *MeerDAG) GetMainChainTip() IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.getMainChainTip()
}

// return the tip of main chain
func (bd *MeerDAG) getMainChainTip() IBlock {
	return bd.instance.GetMainChainTip()
}

// return the main parent in the parents
func (bd *MeerDAG) GetMainParent(parents *IdSet) IBlock {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.instance.GetMainParent(parents)
}

func (bd *MeerDAG) GetMainParentAndList(parents []*hash.Hash) (IBlock, []*hash.Hash) {
	pids := bd.GetIdSet(parents)

	bd.stateLock.Lock()
	mp := bd.instance.GetMainParent(pids)
	bd.stateLock.Unlock()

	ps := []*hash.Hash{mp.GetHash()}
	for _, pt := range parents {
		if pt.IsEqual(mp.GetHash()) {
			continue
		}
		ps = append(ps, pt)
	}
	return mp, ps
}

// return the main parent in the parents
func (bd *MeerDAG) GetMainParentByHashs(parents []*hash.Hash) IBlock {
	pids := bd.GetIdSet(parents)

	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	return bd.instance.GetMainParent(pids)
}

// Return current general description of the whole state of DAG
func (bd *MeerDAG) GetGraphState() *GraphState {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.getGraphState()
}

// Return current general description of the whole state of DAG
func (bd *MeerDAG) getGraphState() *GraphState {
	gs := NewGraphState()
	gs.SetLayer(0)

	tips := bd.getValidTips(false)
	tipsH := []*hash.Hash{}
	for i := 0; i < len(tips); i++ {
		tip := tips[i]
		tipsH = append(tipsH, tip.GetHash())
		if tip.GetLayer() > gs.GetLayer() {
			gs.SetLayer(tip.GetLayer())
		}
	}
	gs.SetTips(tipsH)
	gs.SetTotal(bd.blockTotal)
	gs.SetMainHeight(bd.getMainChainTip().GetHeight())
	gs.SetMainOrder(bd.getMainChainTip().GetOrder())
	return gs
}

// Judging whether block is the virtual tip that it have not future set.
func isVirtualTip(bs *IdSet, futureSet *IdSet, anticone *IdSet, children *IdSet) bool {
	for k := range children.GetMap() {
		if bs.Has(k) {
			return false
		}
		if !futureSet.Has(k) && !anticone.Has(k) {
			return false
		}
	}
	return true
}

// This function is used to GetAnticone recursion
func (bd *MeerDAG) recAnticone(bs *IdSet, futureSet *IdSet, anticone *IdSet, ib IBlock) {
	if bs.Has(ib.GetID()) || anticone.Has(ib.GetID()) {
		return
	}
	children := ib.GetChildren()
	needRecursion := false
	if children == nil || children.Size() == 0 {
		needRecursion = true
	} else {
		needRecursion = isVirtualTip(bs, futureSet, anticone, children)
	}
	if needRecursion {
		if !futureSet.Has(ib.GetID()) {
			anticone.AddPair(ib.GetID(), ib)
		}
		parents := bd.getParents(ib)

		//Because parents can not be empty, so there is no need to judge.
		for _, v := range parents.GetMap() {
			pib := v.(IBlock)
			bd.recAnticone(bs, futureSet, anticone, pib)
		}
	}
}

func (bd *MeerDAG) GetAnticone(b IBlock, exclude *IdSet) *IdSet {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.getAnticone(b, exclude)
}

// This function can get anticone set for an block that you offered in the block dag,If
// the exclude set is not empty,the final result will exclude set that you passed in.
func (bd *MeerDAG) getAnticone(b IBlock, exclude *IdSet) *IdSet {
	futureSet := NewIdSet()
	bd.getFutureSet(futureSet, b)
	anticone := NewIdSet()
	bs := NewIdSet()
	bs.AddPair(b.GetID(), b)
	for _, v := range bd.tips.GetMap() {
		ib := v.(IBlock)
		bd.recAnticone(bs, futureSet, anticone, ib)
	}
	if exclude != nil {
		anticone.Exclude(exclude)
	}
	return anticone
}

// getParentsAnticone
func (bd *MeerDAG) getParentsAnticone(parents *IdSet) *IdSet {
	anticone := NewIdSet()
	for _, v := range bd.tips.GetMap() {
		ib := v.(IBlock)
		bd.recAnticone(parents, NewIdSet(), anticone, ib)
	}
	return anticone
}

// getTreeTips
func getTreeTips(root IBlock, mainsubdag *IdSet, genealogy *IdSet) *IdSet {
	allmainsubdag := mainsubdag.Clone()
	queue := []IBlock{}
	for _, v := range root.GetParents().GetMap() {
		ib := v.(IBlock)
		queue = append(queue, ib)
		if genealogy != nil {
			genealogy.Add(ib.GetID())
		}
	}

	startQueue := queue
	queueSet := NewIdSet()

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if queueSet.Has(cur.GetID()) {
			continue
		}
		queueSet.Add(cur.GetID())

		if allmainsubdag.Has(cur.GetID()) {
			allmainsubdag.AddSet(cur.GetParents())
		}
		if !cur.HasParents() {
			continue
		}
		for _, v := range cur.GetParents().GetMap() {
			ib := v.(IBlock)
			queue = append(queue, ib)
		}

	}

	queue = startQueue
	tips := NewIdSet()
	queueSet.Clean()
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.GetID() == 0 {
			tips.AddPair(cur.GetID(), cur)
			continue
		}
		if queueSet.Has(cur.GetID()) {
			continue
		}
		queueSet.Add(cur.GetID())

		if !allmainsubdag.Has(cur.GetID()) {
			if !cur.HasParents() {
				tips.AddPair(cur.GetID(), cur)
			}
			if genealogy != nil {
				genealogy.Add(cur.GetID())
			}
		}
		if !cur.HasParents() {
			continue
		}
		for _, v := range cur.GetParents().GetMap() {
			ib := v.(IBlock)
			queue = append(queue, ib)
		}
	}

	return tips
}

// getDiffAnticone
func (bd *MeerDAG) getDiffAnticone(b IBlock, verbose bool) *IdSet {
	if b.GetMainParent() == MaxId {
		return nil
	}
	parents := bd.getParents(b)
	if parents == nil || parents.Size() <= 1 {
		return nil
	}
	rootBlock := &Block{id: b.GetID(), hash: *b.GetHash(), parents: NewIdSet(), mainParent: MaxId, layer: b.GetLayer()}
	// find anticone
	anticone := NewIdSet()
	mainsubdag := NewIdSet()
	mainsubdag.Add(0)

	var curMP IBlock

	for _, v := range parents.GetMap() {
		ib := v.(IBlock)
		cur := &Block{id: ib.GetID(), hash: *ib.GetHash(), parents: NewIdSet(), mainParent: MaxId, layer: ib.GetLayer()}
		if ib.GetID() == b.GetMainParent() {
			mainsubdag.Add(ib.GetID())
			curMP = ib
		} else {
			rootBlock.parents.AddPair(cur.GetID(), cur)
			anticone.AddPair(cur.GetID(), cur)
		}
	}

	result := NewIdSet()
	anticoneTips := getTreeTips(rootBlock, mainsubdag, result)

	for anticoneTips.Size() > 0 {

		minTipLayer := uint(math.MaxUint32)
		for _, v := range anticoneTips.GetMap() {
			tb := v.(*Block)
			realib := bd.getBlockById(tb.GetID())
			if realib.HasParents() {
				for _, pv := range bd.getParents(realib).GetMap() {
					pib := pv.(IBlock)
					var cur *Block
					if anticone.Has(pib.GetID()) {
						cur = anticone.Get(pib.GetID()).(*Block)
					} else {
						cur = &Block{id: pib.GetID(), hash: *pib.GetHash(), parents: NewIdSet(), mainParent: MaxId, layer: pib.GetLayer()}
						anticone.AddPair(cur.GetID(), cur)
					}
					tb.parents.AddPair(cur.GetID(), cur)
				}
			}
			if tb.GetLayer() < minTipLayer {
				minTipLayer = tb.GetLayer()
			}
		}

		for curMP != nil {
			if curMP.GetLayer() < minTipLayer {
				break
			}
			mainsubdag.Add(curMP.GetID())
			mainsubdag.AddSet(curMP.(*PhantomBlock).GetBlueDiffAnticone())
			mainsubdag.AddSet(curMP.(*PhantomBlock).GetRedDiffAnticone())
			curMP = bd.getBlockById(curMP.GetMainParent())
		}
		result.Clean()
		anticoneTips = getTreeTips(rootBlock, mainsubdag, result)

		if curMP == nil && anticoneTips.HasOnly(0) {
			break
		}
	}

	//
	if verbose && !result.IsEmpty() {
		optimizeDiffAnt := NewIdSet()
		for k := range result.GetMap() {
			optimizeDiffAnt.AddPair(k, bd.getBlockById(k))
		}
		return optimizeDiffAnt
	}
	return result
}

// GetConfirmations
func (bd *MeerDAG) GetConfirmations(id uint) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	block := bd.getBlockById(id)
	if block == nil {
		return 0
	}
	if block.GetOrder() > bd.getMainChainTip().GetOrder() {
		return 0
	}
	mainTip := bd.getMainChainTip()
	if bd.isOnMainChain(id) {
		return mainTip.GetHeight() - block.GetHeight()
	}
	if !block.HasChildren() {
		return 0
	}
	//
	queue := []IBlock{}
	queue = append(queue, block)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if bd.isOnMainChain(cur.GetID()) {
			return 1 + mainTip.GetHeight() - cur.GetHeight()
		}
		if !cur.HasChildren() {
			continue
		} else {
			childList := bd.getChildren(cur).SortHashList(false)
			for _, v := range childList {
				ib := cur.GetChildren().Get(v).(IBlock)
				queue = append(queue, ib)
			}
		}
	}
	return 0
}

// Checking the layer grap of block
func (bd *MeerDAG) checkLayerGap(parentsNode []IBlock) bool {
	if len(parentsNode) == 0 {
		return false
	}

	pLen := len(parentsNode)
	if pLen == 0 {
		return false
	}
	var gap float64
	if pLen == 1 {
		return true
	} else if pLen == 2 {
		gap = math.Abs(float64(parentsNode[0].GetLayer()) - float64(parentsNode[1].GetLayer()))
	} else {
		var minLayer int64 = -1
		var maxLayer int64 = -1
		for i := 0; i < pLen; i++ {
			parentLayer := int64(parentsNode[i].GetLayer())
			if maxLayer == -1 || parentLayer > maxLayer {
				maxLayer = parentLayer
			}
			if minLayer == -1 || parentLayer < minLayer {
				minLayer = parentLayer
			}
		}
		gap = math.Abs(float64(maxLayer) - float64(minLayer))
	}
	if gap > MaxTipLayerGap {
		log.Error(fmt.Sprintf("Parents gap is %f which is more than %d", gap, MaxTipLayerGap))
		return false
	}

	return true
}

// Checking the sub main chain for the parents of tip
func (bd *MeerDAG) CheckSubMainChainTip(parents []*hash.Hash) error {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	if len(parents) == 0 {
		return fmt.Errorf("Parents is empty")
	}
	mainTip := bd.getMainChainTip()
	if mainTip.GetHash().String() != parents[0].String() {
		return fmt.Errorf("main chain tip is overdue,submit parent:%v , but main tip is :%v\n",
			parents[0].String(), mainTip.GetHash().String())
	}
	for _, pa := range parents {
		ib := bd.getBlock(pa)
		if ib == nil {
			return fmt.Errorf("Parent(%s) is overdue\n", pa.String())
		}
		if ib.HasChildren() {
			return fmt.Errorf("Parent(%s) is not legal tip\n", pa.String())
		}
	}
	return nil
}

// Checking the parents of block legitimacy
func (bd *MeerDAG) checkLegality(parentsNode []IBlock) bool {
	if len(parentsNode) == 0 {
		return false
	}

	pLen := len(parentsNode)
	if pLen == 0 {
		return false
	} else if pLen == 1 {
		return true
	} else {
		parentsSet := NewIdSet()
		for _, v := range parentsNode {
			parentsSet.Add(v.GetID())
		}

		// Belonging to close relatives
		for _, p := range parentsNode {
			if p.HasParents() {
				inSet := p.GetParents().Intersection(parentsSet)
				if !inSet.IsEmpty() {
					return false
				}
			}
			if p.HasChildren() {
				inSet := p.GetChildren().Intersection(parentsSet)
				if !inSet.IsEmpty() {
					return false
				}
			}
		}
	}

	return true
}

// Checking the priority of block legitimacy
func (bd *MeerDAG) checkPriority(parents []IBlock, b IBlockData) bool {
	if b.GetPriority() <= 0 {
		return false
	}
	lowPriNum := 0
	for _, pa := range parents {
		if bd.GetBlockData(pa).GetPriority() <= 1 {
			lowPriNum++
		}
	}
	return b.GetPriority() >= lowPriNum
}

func (bd *MeerDAG) IsHourglass(id uint) bool {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	if !bd.hasBlockById(id) {
		return false
	}
	if !bd.isOnMainChain(id) {
		return false
	}
	block := bd.getBlockById(id)
	if block == nil {
		return false
	}
	if !block.IsOrdered() {
		return false
	}
	//
	queueSet := NewIdSet()
	queue := []IBlock{}
	for _, v := range bd.tips.GetMap() {
		ib := v.(IBlock)
		if !ib.IsOrdered() {
			continue
		}
		queue = append(queue, ib)
		queueSet.Add(ib.GetID())
	}

	num := 0
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.GetID() == id {
			num++
			continue
		}
		if cur.GetLayer() <= block.GetLayer() {
			num++
			continue
		}
		if !cur.HasParents() {
			continue
		}
		for _, v := range bd.getParents(cur).GetMap() {
			ib := v.(IBlock)
			if queueSet.Has(ib.GetID()) || !ib.IsOrdered() {
				continue
			}
			queue = append(queue, ib)
			queueSet.Add(ib.GetID())
		}
	}
	return num == 1
}

func (bd *MeerDAG) GetParentsMaxLayer(parents *IdSet) (uint, bool) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	maxLayer := uint(0)
	for k := range parents.GetMap() {
		ib := bd.getBlockById(k)
		if ib == nil {
			return 0, false
		}
		if maxLayer == 0 || maxLayer < ib.GetLayer() {
			maxLayer = ib.GetLayer()
		}
	}
	return maxLayer, true
}

// GetMaturity
func (bd *MeerDAG) GetMaturity(target uint, views []uint) uint {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	if target == MaxId {
		return 0
	}
	targetBlock := bd.getBlockById(target)
	if targetBlock == nil {
		return 0
	}

	//
	maxLayer := targetBlock.GetLayer()
	queueSet := NewIdSet()
	queue := []IBlock{}
	for _, v := range views {
		ib := bd.getBlockById(v)
		if ib != nil && ib.GetLayer() > targetBlock.GetLayer() {
			queue = append(queue, ib)
			queueSet.Add(ib.GetID())

			if maxLayer < ib.GetLayer() {
				maxLayer = ib.GetLayer()
			}
		}
	}

	connected := false
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.GetID() == target {
			connected = true
			break
		}
		if !cur.HasParents() {
			continue
		}
		if cur.GetLayer() <= targetBlock.GetLayer() {
			continue
		}

		for _, v := range bd.getParents(cur).GetMap() {
			ib := v.(IBlock)
			if queueSet.Has(ib.GetID()) {
				continue
			}
			queue = append(queue, ib)
			queueSet.Add(ib.GetID())
		}
	}

	if connected {
		return maxLayer - targetBlock.GetLayer()
	}
	return 0
}

// Get path intersection from block to main chain.
func (bd *MeerDAG) getMainFork(ib IBlock, backward bool) IBlock {
	if bd.instance.isOnMainChain(ib.GetID()) {
		return ib
	}

	//
	queue := []IBlock{}
	queue = append(queue, ib)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if bd.instance.isOnMainChain(cur.GetID()) {
			return cur
		}

		if backward {
			if !cur.HasChildren() {
				continue
			} else {
				childList := bd.getChildren(cur).SortHashList(false)
				for _, v := range childList {
					ib := cur.GetChildren().Get(v).(IBlock)
					queue = append(queue, ib)
				}
			}
		} else {
			if !cur.HasParents() {
				continue
			} else {
				parentsList := bd.getParents(cur).SortHashList(false)
				for _, v := range parentsList {
					ib := cur.GetParents().Get(v).(IBlock)
					queue = append(queue, ib)
				}
			}
		}
	}

	return nil
}

// MaxParentsPerBlock
func (bd *MeerDAG) getMaxParents() int {
	return bd.instance.getMaxParents()
}

// The main parent concurrency of block
func (bd *MeerDAG) GetMainParentConcurrency(b IBlock) int {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.instance.GetMainParentConcurrency(b)
}

// GetBlockConcurrency : Temporarily use blue set of the past blocks as the criterion
func (bd *MeerDAG) GetBlockConcurrency(h *hash.Hash) (uint, error) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	ib := bd.getBlock(h)
	if ib == nil {
		return 0, fmt.Errorf("No find block")
	}
	return ib.(*PhantomBlock).GetBlueNum(), nil
}

func (bd *MeerDAG) AddToCommit(block IBlock) {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()

	bd.commitBlock.AddPair(block.GetID(), block)
}

// Commit the consensus content to the database for persistence
func (bd *MeerDAG) Commit() error {
	bd.stateLock.Lock()
	defer bd.stateLock.Unlock()
	return bd.commit()
}

// Commit the consensus content to the database for persistence
func (bd *MeerDAG) commit() error {
	needPB := false
	if bd.lastSnapshot.IsValid() {
		needPB = true
	} else if bd.lastSnapshot.block != nil {
		if bd.lastSnapshot.block.GetID() == 0 {
			needPB = true
		}
	}
	ph, ok := bd.instance.(*Phantom)
	if needPB {
		err := DBPutDAGBlockIdByHash(bd.db, bd.lastSnapshot.block)
		if err != nil {
			return err
		}
		for k := range bd.tips.GetMap() {
			if bd.lastSnapshot.tips.Has(k) &&
				k != bd.instance.GetMainChainTipId() &&
				k != bd.lastSnapshot.mainChainTip {
				continue
			}
			err := DBPutDAGTip(bd.db, k, k == bd.instance.GetMainChainTipId())
			if err != nil {
				return err
			}
		}

		for k := range bd.lastSnapshot.tips.GetMap() {
			if bd.tips.Has(k) {
				continue
			}
			err := DBDelDAGTip(bd.db, k)
			if err != nil {
				return err
			}
		}

		if ok {
			for k := range ph.diffAnticone.GetMap() {
				if bd.lastSnapshot.diffAnticone.Has(k) {
					continue
				}
				err = DBPutDiffAnticone(bd.db, k)
				if err != nil {
					return err
				}
			}

			for k := range bd.lastSnapshot.diffAnticone.GetMap() {
				if ph.diffAnticone.Has(k) {
					continue
				}
				err := DBDelDiffAnticone(bd.db, k)
				if err != nil {
					return err
				}
			}
		}
		bd.lastSnapshot.Clean()
	}

	if len(bd.commitOrder) > 0 {
		for order, id := range bd.commitOrder {
			err := DBPutBlockIdByOrder(bd.db, order, id)
			if err != nil {
				log.Error(err.Error())
				return err
			}
		}
		bd.commitOrder = map[uint]uint{}
	}

	if !bd.commitBlock.IsEmpty() {
		for _, v := range bd.commitBlock.GetMap() {
			block, ok := v.(IBlock)
			if !ok {
				return fmt.Errorf("Commit block error\n")
			}
			err := DBPutDAGBlock(bd.db, block)
			if err != nil {
				return err
			}
		}
		bd.commitBlock.Clean()
	}
	if !ok {
		return nil
	}
	err := ph.mainChain.commit()
	if err != nil {
		return err
	}
	bd.optimizeTips()
	return nil
}

func (bd *MeerDAG) Rollback() error {
	if bd.lastSnapshot.IsValid() {
		log.Debug(fmt.Sprintf("Block DAG try to roll back ... ..."))

		block := bd.lastSnapshot.block
		delete(bd.blocks, block.GetID())
		bd.commitBlock.Clean()

		for _, v := range bd.getParents(block).GetMap() {
			parent, ok := v.(IBlock)
			if !ok {
				log.Error(fmt.Sprintf("Can't remove child info for %s", block.GetHash()))
				continue
			}
			parent.RemoveChild(block.GetID())
		}

		bd.blockTotal--
		bd.tips = bd.lastSnapshot.tips
		bd.lastTime = bd.lastSnapshot.lastTime

		if ph, ok := bd.instance.(*Phantom); ok {
			ph.mainChain.tip = bd.lastSnapshot.mainChainTip
			ph.mainChain.genesis = bd.lastSnapshot.mainChainGenesis
			ph.mainChain.commitBlocks.Clean()
			ph.diffAnticone = bd.lastSnapshot.diffAnticone
		}

		if !bd.lastSnapshot.orders.IsEmpty() {
			for _, v := range bd.lastSnapshot.orders.GetMap() {
				boh, ok := v.(*BlockOrderHelp)
				if !ok {
					log.Error("DAG roll back orders type error")
					continue
				}
				boh.Block.SetOrder(boh.OldOrder)
			}
		}
		bd.commitOrder = map[uint]uint{}

		bd.lastSnapshot.Clean()
	} else {
		return fmt.Errorf("No DAG snapshot data for roll back")
	}

	return nil
}

// Just for custom Virtual block
func (bd *MeerDAG) CreateVirtualBlock(data IBlockData) IBlock {
	if _, ok := bd.instance.(*Phantom); !ok {
		return nil
	}
	parents := NewIdSet()
	var maxLayer uint = 0
	var mainParent IBlock
	for k, p := range data.GetParents() {
		ib := bd.GetBlock(p)
		if ib == nil {
			return nil
		}
		if k == 0 {
			mainParent = ib
		}
		parents.AddPair(ib.GetID(), ib)
		if maxLayer == 0 || maxLayer < ib.GetLayer() {
			maxLayer = ib.GetLayer()
		}
	}
	mainParentId := MaxId
	mainHeight := uint(0)
	if mainParent != nil {
		mainParentId = mainParent.GetID()
		mainHeight = mainParent.GetHeight()
	}
	block := Block{id: bd.GetBlockTotal(), hash: *data.GetHash(), parents: parents, layer: maxLayer + 1, mainParent: mainParentId, data: data, height: mainHeight + 1, state: createBlockState(uint64(bd.GetBlockTotal()))}
	return &PhantomBlock{&block, 0, NewIdSet(), NewIdSet()}
}

func (bd *MeerDAG) optimizeReorganizeResult(newOrders *list.List, oldOrders *list.List) {
	if newOrders == nil || oldOrders == nil {
		return
	}
	if newOrders.Len() <= 0 || oldOrders.Len() <= 0 {
		return
	}
	// optimization
	ne := newOrders.Front()
	oe := oldOrders.Front()
	for {
		if ne == nil || oe == nil {
			break
		}
		neNext := ne.Next()
		oeNext := oe.Next()

		neBlock := ne.Value.(IBlock)
		oeBlock := oe.Value.(*BlockOrderHelp)
		if neBlock.GetID() == oeBlock.Block.GetID() && neBlock.GetOrder() == oeBlock.OldOrder {
			newOrders.Remove(ne)
			oldOrders.Remove(oe)
		} else {
			break
		}

		ne = neNext
		oe = oeNext
	}
}

func (bd *MeerDAG) handler() {
	log.Trace("MeerDAG handler start")
	stallTicker := time.NewTicker(params.ActiveNetParams.TargetTimePerBlock * 2)
	defer stallTicker.Stop()

out:
	for {
		select {
		case <-stallTicker.C:
			bd.updateBlockDataCache()
			bd.updateBlockCache()
		case <-bd.quit:
			break out
		}
	}
	bd.wg.Done()
	log.Trace("MeerDAG handler done")
}

func New(dagType string, blockRate float64, db model.DataBase, getBlockData GetBlockData, createBS CreateBlockState, createBSB CreateBlockStateFromBytes) *MeerDAG {
	createBlockState = createBS
	createBlockStateFromBytes = createBSB
	md := &MeerDAG{
		quit:           make(chan struct{}),
		blockDataCache: map[uint]time.Time{},
		minCacheSize:   MinBlockPruneSize,
		minBDCacheSize: MinBlockDataCache,
	}
	md.init(dagType, blockRate, db, getBlockData)
	return md
}
