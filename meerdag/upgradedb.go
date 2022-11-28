package meerdag

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"github.com/schollz/progressbar/v3"
	"io"
)

// update db to new version
func (bd *MeerDAG) UpgradeDB(dbTx database.Tx, mainTip *hash.Hash, total uint64, genesis *hash.Hash, fortips bool, interrupt <-chan struct{}) error {
	if fortips {
		bucket := dbTx.Metadata().Bucket(DAGTipsBucketName)
		cursor := bucket.Cursor()
		if cursor.First() {
			return fmt.Errorf("Data format error: already exists tips")
		}
	}
	log.Info(fmt.Sprintf("Start upgrade MeerDAG🛠 (total=%d mainTip=%s)", total, mainTip.String()))
	//
	var bar *progressbar.ProgressBar
	logLvl := l.Glogger().GetVerbosity()
	bar = progressbar.Default(int64(total), "MeerDAG:")
	l.Glogger().Verbosity(l.LvlCrit)
	defer func() {
		bar.Finish()
		l.Glogger().Verbosity(logLvl)
	}()
	//
	blocks := map[uint]IBlock{}
	var tips *IdSet
	var mainTipBlock IBlock

	getBlockById := func(id uint) IBlock {
		if id == MaxId {
			return nil
		}
		block, ok := blocks[id]
		if !ok {
			return nil
		}
		return block
	}

	updateTips := func(b IBlock) {
		if tips == nil {
			tips = NewIdSet()
			tips.AddPair(b.GetID(), b)
			return
		}
		for k, v := range tips.GetMap() {
			block := v.(IBlock)
			if block.HasChildren() {
				tips.Remove(k)
			}
		}
		tips.AddPair(b.GetID(), b)
	}
	diffAnticone := NewIdSet()
	for i := uint(0); i < uint(total); i++ {
		bar.Add(1)
		if system.InterruptRequested(interrupt) {
			return fmt.Errorf("interrupt upgrade database")
		}
		block := OldBlock{id: i}
		ib := &OldPhantomBlock{&block, 0, NewIdSet(), NewIdSet()}
		err := DBGetDAGBlock(dbTx, ib)
		if err != nil {
			if err.(*DAGError).IsEmpty() {
				continue
			}
			return err
		}
		if i == 0 && !ib.GetHash().IsEqual(genesis) {
			return fmt.Errorf("genesis data mismatch")
		}
		if ib.HasParents() {
			parentsSet := NewIdSet()
			for k := range ib.GetParents().GetMap() {
				parent := getBlockById(k)
				parentsSet.AddPair(k, parent)
				parent.AddChild(ib)
			}
			ib.GetParents().Clean()
			ib.GetParents().AddSet(parentsSet)
		}
		blocks[ib.GetID()] = ib
		if !ib.IsOrdered() {
			diffAnticone.AddPair(ib.GetID(), ib)
		}
		if fortips {
			updateTips(ib)
			if ib.GetHash().IsEqual(mainTip) {
				mainTipBlock = ib
			}
		}
	}
	bar = progressbar.Default(int64(len(blocks)), "MeerDAG:")
	for _, ib := range blocks {
		bar.Add(1)
		if system.InterruptRequested(interrupt) {
			return fmt.Errorf("interrupt upgrade database")
		}
		block := ib.(*OldPhantomBlock).toPhantomBlock()
		err := DBPutDAGBlock(dbTx, block)
		if err != nil {
			return err
		}
	}
	blocks = nil
	if !diffAnticone.IsEmpty() {
		for id := range diffAnticone.GetMap() {
			err := DBPutDiffAnticone(dbTx, id)
			if err != nil {
				return err
			}
		}
		log.Info(fmt.Sprintf("Upgrade diffAnticone size:%d", diffAnticone.Size()))
		diffAnticone.Clean()
	}
	if fortips {
		if mainTipBlock == nil || tips == nil || tips.IsEmpty() || !tips.Has(mainTipBlock.GetID()) {
			return fmt.Errorf("Main chain tip error")
		}

		for k := range tips.GetMap() {
			err := DBPutDAGTip(dbTx, k, k == mainTipBlock.GetID())
			if err != nil {
				return err
			}
		}
		log.Info(fmt.Sprintf("End upgrade MeerDAG.tips🛠:bridging tips num(%d)", tips.Size()))
		tips.Clean()
	}
	return nil
}

// old block
type OldPhantomBlock struct {
	*OldBlock
	blueNum uint

	blueDiffAnticone *IdSet
	redDiffAnticone  *IdSet
}

func (pb *OldPhantomBlock) IsBluer(other *OldPhantomBlock) bool {
	if pb.blueNum > other.blueNum {
		return true
	} else if pb.blueNum == other.blueNum {
		if pb.GetData().GetPriority() > other.GetData().GetPriority() {
			return true
		} else if pb.GetData().GetPriority() == other.GetData().GetPriority() {
			if pb.GetHash().String() < other.GetHash().String() {
				return true
			}
		}
	}
	return false
}

// encode
func (pb *OldPhantomBlock) Encode(w io.Writer) error {
	err := pb.OldBlock.Encode(w)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint32(pb.blueNum))
	if err != nil {
		return err
	}

	// blueDiffAnticone
	blueDiffAnticone := []uint{}
	if pb.GetBlueDiffAnticoneSize() > 0 {
		blueDiffAnticone = pb.blueDiffAnticone.List()
	}
	blueDiffAnticoneSize := len(blueDiffAnticone)
	err = s.WriteElements(w, uint32(blueDiffAnticoneSize))
	if err != nil {
		return err
	}
	for i := 0; i < blueDiffAnticoneSize; i++ {
		err = s.WriteElements(w, uint32(blueDiffAnticone[i]))
		if err != nil {
			return err
		}
		order := pb.blueDiffAnticone.Get(blueDiffAnticone[i]).(uint)
		err = s.WriteElements(w, uint32(order))
		if err != nil {
			return err
		}
	}
	// redDiffAnticone
	redDiffAnticone := []uint{}
	if pb.redDiffAnticone != nil && pb.redDiffAnticone.Size() > 0 {
		redDiffAnticone = pb.redDiffAnticone.List()
	}
	redDiffAnticoneSize := len(redDiffAnticone)
	err = s.WriteElements(w, uint32(redDiffAnticoneSize))
	if err != nil {
		return err
	}
	for i := 0; i < redDiffAnticoneSize; i++ {
		err = s.WriteElements(w, uint32(redDiffAnticone[i]))
		if err != nil {
			return err
		}
		order := pb.redDiffAnticone.Get(redDiffAnticone[i]).(uint)
		err = s.WriteElements(w, uint32(order))
		if err != nil {
			return err
		}
	}
	return nil
}

// decode
func (pb *OldPhantomBlock) Decode(r io.Reader) error {
	err := pb.OldBlock.Decode(r)
	if err != nil {
		return err
	}

	var blueNum uint32
	err = s.ReadElements(r, &blueNum)
	if err != nil {
		return err
	}
	pb.blueNum = uint(blueNum)

	// blueDiffAnticone
	var blueDiffAnticoneSize uint32
	err = s.ReadElements(r, &blueDiffAnticoneSize)
	if err != nil {
		return err
	}
	if blueDiffAnticoneSize > 0 {
		for i := uint32(0); i < blueDiffAnticoneSize; i++ {
			var bda uint32
			err := s.ReadElements(r, &bda)
			if err != nil {
				return err
			}

			var order uint32
			err = s.ReadElements(r, &order)
			if err != nil {
				return err
			}

			pb.AddPairBlueDiffAnticone(uint(bda), uint(order))
		}
	}

	// redDiffAnticone
	var redDiffAnticoneSize uint32
	err = s.ReadElements(r, &redDiffAnticoneSize)
	if err != nil {
		return err
	}
	if redDiffAnticoneSize > 0 {
		for i := uint32(0); i < redDiffAnticoneSize; i++ {
			var bda uint32
			err := s.ReadElements(r, &bda)
			if err != nil {
				return err
			}
			var order uint32
			err = s.ReadElements(r, &order)
			if err != nil {
				return err
			}

			pb.AddPairRedDiffAnticone(uint(bda), uint(order))
		}
	}

	return nil
}

func (pb *OldPhantomBlock) GetBlueDiffAnticoneSize() int {
	if pb.blueDiffAnticone == nil {
		return 0
	}
	return pb.blueDiffAnticone.Size()
}

func (pb *OldPhantomBlock) AddPairBlueDiffAnticone(id uint, order uint) {
	if pb.blueDiffAnticone == nil {
		pb.blueDiffAnticone = NewIdSet()
	}
	pb.blueDiffAnticone.AddPair(id, order)
}

func (pb *OldPhantomBlock) AddPairRedDiffAnticone(id uint, order uint) {
	if pb.redDiffAnticone == nil {
		pb.redDiffAnticone = NewIdSet()
	}
	pb.redDiffAnticone.AddPair(id, order)
}

func (pb *OldPhantomBlock) toPhantomBlock() *PhantomBlock {
	return &PhantomBlock{
		Block:            &Block{id: pb.id, hash: pb.hash, parents: pb.parents, children: pb.children, mainParent: pb.mainParent, weight: pb.weight, order: pb.order, layer: pb.layer, height: pb.height, status: pb.status, data: pb.data},
		blueNum:          pb.blueNum,
		blueDiffAnticone: pb.blueDiffAnticone,
		redDiffAnticone:  pb.redDiffAnticone,
	}
}

type OldBlock struct {
	id       uint
	hash     hash.Hash
	parents  *IdSet
	children *IdSet

	mainParent uint
	weight     uint64
	order      uint
	layer      uint
	height     uint
	status     BlockStatus

	data IBlockData
}

// Return block ID
func (b *OldBlock) GetID() uint {
	return b.id
}

func (b *OldBlock) SetID(id uint) {
	b.id = id
}

func (b *OldBlock) GetHash() *hash.Hash {
	return &b.hash
}

func (b *OldBlock) AddParent(parent IBlock) {
	if b.parents == nil {
		b.parents = NewIdSet()
	}
	b.parents.AddPair(parent.GetID(), parent)
}

func (b *OldBlock) GetParents() *IdSet {
	return b.parents
}

func (b *OldBlock) GetMainParent() uint {
	return b.mainParent
}

func (b *OldBlock) HasParents() bool {
	if b.parents == nil {
		return false
	}
	if b.parents.IsEmpty() {
		return false
	}
	return true
}

func (b *OldBlock) GetForwardParent() *Block {
	if b.parents == nil || b.parents.IsEmpty() {
		return nil
	}
	var result *Block = nil
	for _, v := range b.parents.GetMap() {
		parent := v.(*Block)
		if result == nil || parent.GetOrder() < result.GetOrder() {
			result = parent
		}
	}
	return result
}

func (b *OldBlock) GetBackParent() *Block {
	if b == nil || b.parents == nil || b.parents.IsEmpty() {
		return nil
	}
	var result *Block = nil
	for _, v := range b.parents.GetMap() {
		parent := v.(*Block)
		if result == nil || parent.GetOrder() > result.GetOrder() {
			result = parent
		}
	}
	return result
}

func (b *OldBlock) AddChild(child IBlock) {
	if b.children == nil {
		b.children = NewIdSet()
	}
	b.children.AddPair(child.GetID(), child)
}

func (b *OldBlock) GetChildren() *IdSet {
	return b.children
}

func (b *OldBlock) HasChildren() bool {
	if b.children == nil {
		return false
	}
	if b.children.IsEmpty() {
		return false
	}
	return true
}

func (b *OldBlock) RemoveChild(child uint) {
	if !b.HasChildren() {
		return
	}
	b.children.Remove(child)
}

func (b *OldBlock) SetWeight(weight uint64) {
	b.weight = weight
}

func (b *OldBlock) GetWeight() uint64 {
	return b.weight
}

// Setting the layer of block
func (b *OldBlock) SetLayer(layer uint) {
	b.layer = layer
}

// Acquire the layer of block
func (b *OldBlock) GetLayer() uint {
	return b.layer
}

// Setting the order of block
func (b *OldBlock) SetOrder(o uint) {
	b.order = o
}

// Acquire the order of block
func (b *OldBlock) GetOrder() uint {
	return b.order
}

// IsOrdered
func (b *OldBlock) IsOrdered() bool {
	return b.GetOrder() != MaxBlockOrder
}

// Setting the height of block in main chain
func (b *OldBlock) SetHeight(h uint) {
	b.height = h
}

// Acquire the height of block in main chain
func (b *OldBlock) GetHeight() uint {
	return b.height
}

// encode
func (b *OldBlock) Encode(w io.Writer) error {
	err := s.WriteElements(w, uint32(b.id))
	if err != nil {
		return err
	}
	err = s.WriteElements(w, &b.hash)
	if err != nil {
		return err
	}
	// parents
	parents := []uint{}
	if b.HasParents() {
		parents = b.parents.List()
	}
	parentsSize := len(parents)
	err = s.WriteElements(w, uint32(parentsSize))
	if err != nil {
		return err
	}
	for i := 0; i < parentsSize; i++ {
		err = s.WriteElements(w, uint32(parents[i]))
		if err != nil {
			return err
		}
	}
	// mainParent
	mainParent := uint32(MaxId)
	if b.mainParent != MaxId {
		mainParent = uint32(b.mainParent)
	}
	err = s.WriteElements(w, mainParent)
	if err != nil {
		return err
	}

	err = s.WriteElements(w, uint64(b.weight))
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint32(b.order))
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint32(b.layer))
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint32(b.height))
	if err != nil {
		return err
	}
	return s.WriteElements(w, byte(b.status))
}

// decode
func (b *OldBlock) Decode(r io.Reader) error {
	var id uint32
	err := s.ReadElements(r, &id)
	if err != nil {
		return err
	}
	b.id = uint(id)

	err = s.ReadElements(r, &b.hash)
	if err != nil {
		return err
	}
	// parents
	var parentsSize uint32
	err = s.ReadElements(r, &parentsSize)
	if err != nil {
		return err
	}
	if parentsSize > 0 {
		b.parents = NewIdSet()
		for i := uint32(0); i < parentsSize; i++ {
			var parent uint32
			err := s.ReadElements(r, &parent)
			if err != nil {
				return err
			}
			b.parents.Add(uint(parent))
		}
	}
	// mainParent
	var mainParent uint32
	err = s.ReadElements(r, &mainParent)
	if err != nil {
		return err
	}
	b.mainParent = uint(mainParent)

	var weight uint64
	err = s.ReadElements(r, &weight)
	if err != nil {
		return err
	}
	b.weight = uint64(weight)

	var order uint32
	err = s.ReadElements(r, &order)
	if err != nil {
		return err
	}
	b.order = uint(order)

	var layer uint32
	err = s.ReadElements(r, &layer)
	if err != nil {
		return err
	}
	b.layer = uint(layer)

	var height uint32
	err = s.ReadElements(r, &height)
	if err != nil {
		return err
	}
	b.height = uint(height)

	var status byte
	err = s.ReadElements(r, &status)
	if err != nil {
		return err
	}
	b.status = BlockStatus(status)
	return nil
}

// SetStatus
func (b *OldBlock) SetStatus(status BlockStatus) {
	b.status = status
}

func (b *OldBlock) GetStatus() BlockStatus {
	return b.status
}

func (b *OldBlock) SetStatusFlags(flags BlockStatus) {
	b.status |= flags
}

func (b *OldBlock) UnsetStatusFlags(flags BlockStatus) {
	b.status &^= flags
}

func (b *OldBlock) GetData() IBlockData {
	return b.data
}

func (b *OldBlock) SetData(data IBlockData) {
	b.data = data
}

func (b *OldBlock) IsLoaded() bool {
	return b.data != nil
}

func (b *OldBlock) Valid() {
	b.UnsetStatusFlags(StatusInvalid)
}

func (b *OldBlock) Invalid() {
	b.SetStatusFlags(StatusInvalid)
}

func (b *OldBlock) AttachParent(ib IBlock) {
	if !b.HasParents() {
		return
	}
	if !b.parents.Has(ib.GetID()) {
		return
	}
	b.AddParent(ib)
}

func (b *OldBlock) DetachParent(ib IBlock) {
	if !b.HasParents() {
		return
	}
	if !b.parents.Has(ib.GetID()) {
		return
	}
	b.parents.Add(ib.GetID())
}

func (b *OldBlock) AttachChild(ib IBlock) {
	if !b.HasChildren() {
		return
	}
	if !b.children.Has(ib.GetID()) {
		return
	}
	b.AddChild(ib)
}

func (b *OldBlock) DetachChild(ib IBlock) {
	if !b.HasChildren() {
		return
	}
	if !b.children.Has(ib.GetID()) {
		return
	}
	b.children.Add(ib.GetID())
}
