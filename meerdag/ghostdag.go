package meerdag

import (
	"container/list"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	cmodel "github.com/Qitmeer/qng/consensus/model"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/meerdag/anticone"
	"github.com/Qitmeer/qng/meerdag/ghostdag"
	"github.com/Qitmeer/qng/meerdag/ghostdag/model"
	"github.com/Qitmeer/qng/params"
	"io"
)

type GhostDAG struct {
	// The general foundation framework of DAG
	bd *MeerDAG

	// The block anticone size is all in the DAG which did not reference it and
	// were not referenced by it.
	anticoneSize int

	algorithm *ghostdag.GhostDAG

	bgdDatas map[hash.Hash]*model.BlockGHOSTDAGData

	mainChainTip IBlock

	virtualBlock *Block
}

func (gd *GhostDAG) GetName() string {
	return GHOSTDAG
}

func (gd *GhostDAG) Init(bd *MeerDAG) bool {
	gd.bgdDatas = map[hash.Hash]*model.BlockGHOSTDAGData{}
	gd.bd = bd
	gd.anticoneSize = anticone.GetSize(anticone.BlockDelay, bd.blockRate, anticone.SecurityLevel)
	if log != nil {
		log.Info(fmt.Sprintf("anticone size:%d", gd.anticoneSize))
	}
	gd.algorithm = ghostdag.New(nil, gd, gd, gd, model.KType(gd.anticoneSize), params.ActiveNetParams.GenesisHash)

	//vb
	gd.virtualBlock = &Block{hash: model.VirtualBlockHash, layer: 0, mainParent: MaxId, parents: NewIdSet(), state: CreateMockBlockState(uint64(MaxId))}
	return true
}

// Add a block
func (gd *GhostDAG) AddBlock(ib IBlock) (*list.List, *list.List) {
	if ib.GetID() == 0 {
		if !gd.algorithm.GenesisHash().IsEqual(ib.GetHash()) {
			gd.algorithm.SetGenesisHash(ib.GetHash())
		}
	}
	err := gd.algorithm.GHOSTDAG(nil, ib.GetHash())
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	ghostdagData, err := gd.Get(nil, nil, ib.GetHash(), false)
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	ib.(*Block).mainParent = gd.bd.getBlockId(ghostdagData.SelectedParent())

	gd.virtualBlock.parents.Clean()
	gd.virtualBlock.parents.AddSet(gd.bd.tips)
	//
	err = gd.algorithm.GHOSTDAG(nil, gd.virtualBlock.GetHash())
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	vghostdagData, err := gd.Get(nil, nil, gd.virtualBlock.GetHash(), false)
	if err != nil {
		log.Error(err.Error())
		return nil, nil
	}
	gd.virtualBlock.mainParent = gd.bd.getBlockId(vghostdagData.SelectedParent())

	gd.virtualBlock.SetOrder(MaxBlockOrder)
	gd.mainChainTip = gd.bd.getBlockById(gd.virtualBlock.GetMainParent())
	return nil, nil
}

// Build self block
func (gd *GhostDAG) CreateBlock(b *Block) IBlock {
	return b
}

// If the successor return nil, the underlying layer will use the default tips list.
func (gd *GhostDAG) GetTipsList() []IBlock {
	return nil
}

// Query whether a given block is on the main chain.
func (gd *GhostDAG) isOnMainChain(id uint) bool {
	b := gd.bd.getBlockById(id)
	for cur := gd.mainChainTip; cur != nil; cur = gd.bd.getBlockById(cur.GetMainParent()) {
		if cur.GetHash().IsEqual(b.GetHash()) {
			return true
		}
	}
	return false
}

// return the tip of main chain
func (gd *GhostDAG) GetMainChainTip() IBlock {
	return gd.mainChainTip
}

func (gd *GhostDAG) GetMainChainTipId() uint {
	if gd.mainChainTip == nil {
		return MaxId
	}
	return gd.mainChainTip.GetID()
}

// return the main parent in the parents
func (gd *GhostDAG) GetMainParent(parents *IdSet) IBlock {
	virtualBlockParents := []*hash.Hash{}
	for id := range parents.GetMap() {
		virtualBlockParents = append(virtualBlockParents, gd.bd.getBlockById(id).GetHash())
	}
	mainParent, err := gd.algorithm.FindSelectedParent(nil, virtualBlockParents)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return gd.bd.getBlock(mainParent)
}

// encode
func (gd *GhostDAG) Encode(w io.Writer) error {
	err := s.WriteElements(w, uint32(gd.anticoneSize))
	if err != nil {
		return err
	}
	return nil
}

// decode
func (gd *GhostDAG) Decode(r io.Reader) error {
	var anticoneSize uint32
	err := s.ReadElements(r, &anticoneSize)
	if err != nil {
		return err
	}
	if anticoneSize != uint32(gd.anticoneSize) {
		return fmt.Errorf("The anticoneSize (%d) is not the same. (%d)", gd.anticoneSize, anticoneSize)
	}
	return nil
}

// load
func (gd *GhostDAG) Load() error {
	return nil
}

func (gd *GhostDAG) GetBlues(parents *IdSet) uint {
	return 0
}

func (gd *GhostDAG) IsBlue(id uint) bool {
	return false
}

// IsDAG
func (gd *GhostDAG) IsDAG(parents []IBlock) bool {
	if len(parents) == 0 {
		return false
	} else if len(parents) == 1 {
		return true
	}
	return true
}

// The main parent concurrency of block
func (gd *GhostDAG) GetMainParentConcurrency(b IBlock) int {
	return 0
}

func (gd *GhostDAG) getMaxParents() int {
	return 0
}

func (gd *GhostDAG) GetBlueSet() *IdSet {
	if gd.mainChainTip == nil {
		return nil
	}
	blueSet := NewIdSet()
	for cur := IBlock(gd.virtualBlock); cur != nil; cur = gd.bd.getBlockById(cur.GetMainParent()) {
		gdd, err := gd.Get(nil, nil, cur.GetHash(), true)
		if err != nil {
			log.Error(err.Error())
			return nil
		}
		for _, v := range gdd.MergeSetBlues() {
			ib := gd.bd.getBlock(v)
			if ib == nil {
				log.Error(fmt.Sprintf("no block:%s", v))
				return nil
			}
			blueSet.AddPair(ib.GetID(), ib)
		}
	}
	return blueSet
}

// It is only used to simulate the tags of all sequences, and the algorithm itself is very inefficient
func (gd *GhostDAG) UpdateOrders() error {
	if gd.virtualBlock.IsOrdered() {
		return nil
	}
	mainChains := []IBlock{}
	for cur := IBlock(gd.virtualBlock); cur != nil; cur = gd.bd.getBlockById(cur.GetMainParent()) {
		mainChains = append(mainChains, cur)
	}
	curOrder := uint(0)
	for i := len(mainChains) - 1; i >= 0; i-- {
		sms, err := gd.algorithm.GetSortedMergeSet(nil, mainChains[i].GetHash())
		if err != nil {
			return err
		}
		for _, v := range sms {
			block := gd.bd.getBlock(v)
			if block.GetID() == mainChains[i].GetMainParent() {
				continue
			}
			block.SetOrder(curOrder)
			gd.bd.commitOrder[curOrder] = block.GetID()
			curOrder++
		}
		mainChains[i].SetOrder(curOrder)
		gd.bd.commitOrder[curOrder] = mainChains[i].GetID()
		curOrder++

	}
	return gd.bd.commit()
}

// ---------------
// implementation
func (gd *GhostDAG) BlockHeader(dbContext model.DBReader, stagingArea *cmodel.StagingArea, blockHash *hash.Hash) (model.BlockHeader, error) {
	return ghostdag.NewBlockHeader(params.ActiveNetParams.GenesisBlock.Block().Header.Difficulty, params.ActiveNetParams.GenesisBlock.Block().Header.Pow), nil
}

func (gd *GhostDAG) HasBlockHeader(dbContext model.DBReader, stagingArea *cmodel.StagingArea, blockHash *hash.Hash) (bool, error) {
	return true, nil
}

func (gd *GhostDAG) BlockHeaders(dbContext model.DBReader, stagingArea *cmodel.StagingArea, blockHashes []*hash.Hash) ([]model.BlockHeader, error) {
	bhs := []model.BlockHeader{}
	for _, h := range blockHashes {
		bh, err := gd.BlockHeader(dbContext, stagingArea, h)
		if err != nil {
			return nil, err
		}
		bhs = append(bhs, bh)
	}
	return bhs, nil
}

func (gd *GhostDAG) Delete(stagingArea *cmodel.StagingArea, blockHash *hash.Hash) {
	panic("implement me")
}

func (gd *GhostDAG) Count(stagingArea *cmodel.StagingArea) uint64 {
	return uint64(gd.bd.blockTotal)
}

func (gd *GhostDAG) Parents(stagingArea *cmodel.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error) {
	var ib IBlock
	if blockHash.IsEqual(&model.VirtualBlockHash) {
		ib = gd.virtualBlock
	} else {
		ib = gd.bd.getBlock(blockHash)
	}
	if ib == nil {
		return nil, fmt.Errorf("No block:%s", blockHash)
	}
	if !ib.HasParents() {
		return nil, nil
	}
	ps := []*hash.Hash{}
	for _, v := range ib.GetParents().GetMap() {
		ps = append(ps, v.(IBlock).GetHash())
	}
	return ps, nil
}

func (gd *GhostDAG) Children(stagingArea *cmodel.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error) {
	return nil, nil
}

func (gd *GhostDAG) IsParentOf(stagingArea *cmodel.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	return false, nil
}

func (gd *GhostDAG) IsChildOf(stagingArea *cmodel.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	return false, nil
}

func (gd *GhostDAG) IsAncestorOf(stagingArea *cmodel.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	blockBParents, err := gd.Parents(stagingArea, blockHashB)
	if err != nil {
		return false, err
	}
	if len(blockBParents) <= 0 {
		return false, nil
	}

	for _, parentOfB := range blockBParents {
		if parentOfB.IsEqual(blockHashA) {
			return true, nil
		}
	}

	for _, parentOfB := range blockBParents {
		isAncestorOf, err := gd.IsAncestorOf(stagingArea, blockHashA, parentOfB)
		if err != nil {
			return false, err
		}
		if isAncestorOf {
			return true, nil
		}
	}
	return false, nil
}

func (gd *GhostDAG) IsAncestorOfAny(stagingArea *cmodel.StagingArea, blockHash *hash.Hash, potentialDescendants []*hash.Hash) (bool, error) {
	return false, nil
}

func (gd *GhostDAG) IsAnyAncestorOf(stagingArea *cmodel.StagingArea, potentialAncestors []*hash.Hash, blockHash *hash.Hash) (bool, error) {
	return false, nil
}

func (gd *GhostDAG) IsInSelectedParentChainOf(stagingArea *cmodel.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	return false, nil
}

func (gd *GhostDAG) ChildInSelectedParentChainOf(stagingArea *cmodel.StagingArea, lowHash, highHash *hash.Hash) (*hash.Hash, error) {
	return nil, nil
}

func (gd *GhostDAG) SetParents(stagingArea *cmodel.StagingArea, blockHash *hash.Hash, parentHashes []*hash.Hash) error {
	return nil
}

func (gd *GhostDAG) Stage(stagingArea *cmodel.StagingArea, blockHash *hash.Hash, blockGHOSTDAGData *model.BlockGHOSTDAGData, isTrustedData bool) {
	gd.bgdDatas[*blockHash] = blockGHOSTDAGData
}

func (gd *GhostDAG) IsStaged(stagingArea *cmodel.StagingArea) bool {
	return true
}

func (gd *GhostDAG) Get(dbContext model.DBReader, stagingArea *cmodel.StagingArea, blockHash *hash.Hash, isTrustedData bool) (*model.BlockGHOSTDAGData, error) {
	v, ok := gd.bgdDatas[*blockHash]
	if ok {
		return v, nil
	}
	return nil, nil
}

func (gd *GhostDAG) UnstageAll(stagingArea *cmodel.StagingArea) {
}
