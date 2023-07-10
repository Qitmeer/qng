package meerdag

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/forks"
	"github.com/Qitmeer/qng/consensus/model"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/database/legacydb"
	l "github.com/Qitmeer/qng/log"
	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/Qitmeer/qng/meerevm/eth"
	"github.com/Qitmeer/qng/meerevm/meer"
	"github.com/Qitmeer/qng/params"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"io"
)

// update db to new version
func (bd *MeerDAG) UpgradeDB(db legacydb.DB, mainTip *hash.Hash, total uint64, genesis *hash.Hash, interrupt <-chan struct{}, dbFetchBlockByHash func(dbTx legacydb.Tx, hash *hash.Hash) (*types.SerializedBlock, error), isDuplicateTx func(dbTx legacydb.Tx, txid *hash.Hash, blockHash *hash.Hash) bool, evmbc *core.BlockChain, edb ethdb.Database) error {
	log.Info(fmt.Sprintf("Start upgrade MeerDAG🛠 (total=%d mainTip=%s)", total, mainTip.String()))
	//
	mainTipBlock := getOldBlock(db, mainTip)
	//
	var bar *progressbar.ProgressBar
	logLvl := l.Glogger().GetVerbosity()
	bar = progressbar.Default(int64(mainTipBlock.GetOrder()), "MeerDAG:")
	l.Glogger().Verbosity(l.LvlCrit)
	defer func() {
		bar.Finish()
		l.Glogger().Verbosity(logLvl)
	}()
	//
	evmGenesis := evmbc.GetHeaderByNumber(0)
	if evmGenesis == nil {
		return fmt.Errorf("No evm data")
	} else {
		log.Info("EVM genesis", "hash", evmGenesis.Hash().String())
	}
	curEVM := evmGenesis
	var prev model.BlockState
	for i := uint(0); i <= mainTipBlock.GetOrder(); i++ {
		bar.Add(1)
		if system.InterruptRequested(interrupt) {
			return fmt.Errorf("interrupt upgrade database:Data corruption caused by exiting midway")
		}
		var ib IBlock
		if i == mainTipBlock.GetOrder() {
			ib = mainTipBlock
		} else {
			ib = getOldBlockByOrder(db, i)
		}

		bs := createBlockState(uint64(ib.GetID()))
		opb := ib.(*OldPhantomBlock)
		opb.state = bs
		//
		bs.SetOrder(uint64(ib.GetOrder()))
		bs.SetWeight(opb.GetWeight())
		if opb.status.KnownInvalid() {
			bs.Invalid()
		} else {
			bs.Valid()
		}
		// evm
		if i == 0 {
			curEVM = evmGenesis
			bs.SetEVM(curEVM)
		} else {
			// dups
			var block *types.SerializedBlock
			err := db.View(func(dbTx legacydb.Tx) error {
				var e error
				block, e = dbFetchBlockByHash(dbTx, ib.GetHash())
				return e
			})
			if err != nil {
				return err
			}
			txs := block.Transactions()
			for _, tx := range txs {
				db.View(func(dbTx legacydb.Tx) error {
					tx.IsDuplicate = isDuplicateTx(dbTx, tx.Hash(), block.Hash())
					return nil
				})
			}
			//
			if forks.IsBeforeMeerEVMForkHeight(int64(ib.GetHeight())) {
				curEVM = evmGenesis
			} else {
				number := getBlockNumber(edb, block.Hash())
				if number != 0 {
					header := evmbc.GetHeaderByNumber(number)
					if header == nil {
						return fmt.Errorf("No block in number:%d", number)
					}
					curEVM = header
				}
			}

			bs.Update(block, prev, curEVM)
		}

		//
		npb := opb.toPhantomBlock()
		err := db.Update(func(dbTx legacydb.Tx) error {
			return DBPutDAGBlock(dbTx, npb)
		})
		if err != nil {
			return err
		}
		//
		prev = bs
	}
	log.Info("End upgrade MeerDAG🛠", "state root", mainTipBlock.GetState().Root().String(), "mainTip", mainTipBlock.GetHash().String(), "mainOrder", mainTipBlock.GetOrder())
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

// GetBlueNum
func (pb *OldPhantomBlock) GetBlueNum() uint {
	return pb.blueNum
}

func (pb *OldPhantomBlock) GetBlueDiffAnticone() *IdSet {
	return pb.blueDiffAnticone
}

func (pb *OldPhantomBlock) GetRedDiffAnticone() *IdSet {
	return pb.redDiffAnticone
}

func (pb *OldPhantomBlock) GetBlueDiffAnticoneSize() int {
	if pb.blueDiffAnticone == nil {
		return 0
	}
	return pb.blueDiffAnticone.Size()
}

func (pb *OldPhantomBlock) GetRedDiffAnticoneSize() int {
	if pb.redDiffAnticone == nil {
		return 0
	}
	return pb.redDiffAnticone.Size()
}

func (pb *OldPhantomBlock) GetDiffAnticoneSize() int {
	return pb.GetBlueDiffAnticoneSize() + pb.GetRedDiffAnticoneSize()
}

func (pb *OldPhantomBlock) AddBlueDiffAnticone(id uint) {
	if pb.blueDiffAnticone == nil {
		pb.blueDiffAnticone = NewIdSet()
	}
	pb.blueDiffAnticone.Add(id)
}

func (pb *OldPhantomBlock) AddRedDiffAnticone(id uint) {
	if pb.redDiffAnticone == nil {
		pb.redDiffAnticone = NewIdSet()
	}
	pb.redDiffAnticone.Add(id)
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

func (pb *OldPhantomBlock) HasBlueDiffAnticone(id uint) bool {
	if pb.blueDiffAnticone == nil {
		return false
	}
	return pb.blueDiffAnticone.Has(id)
}

func (pb *OldPhantomBlock) HasRedDiffAnticone(id uint) bool {
	if pb.redDiffAnticone == nil {
		return false
	}
	return pb.redDiffAnticone.Has(id)
}

func (pb *OldPhantomBlock) CleanDiffAnticone() {
	if pb.blueDiffAnticone != nil {
		pb.blueDiffAnticone.Clean()
	}
	if pb.redDiffAnticone != nil {
		pb.redDiffAnticone.Clean()
	}
}

func (pb *OldPhantomBlock) toPhantomBlock() *PhantomBlock {
	return &PhantomBlock{
		Block:            &Block{id: pb.id, hash: pb.hash, parents: pb.parents, children: pb.children, mainParent: pb.mainParent, layer: pb.layer, height: pb.height, data: pb.data, state: pb.state},
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
	status     model.BlockStatus

	data  IBlockData
	state model.BlockState
}

// Return block ID
func (b *OldBlock) GetID() uint {
	return b.id
}

func (b *OldBlock) SetID(id uint) {
	b.id = id
}

// Return the hash of block. It will be a pointer.
func (b *OldBlock) GetHash() *hash.Hash {
	return &b.hash
}

func (b *OldBlock) AddParent(parent IBlock) {
	if b.parents == nil {
		b.parents = NewIdSet()
	}
	b.parents.AddPair(parent.GetID(), parent)
}

func (b *OldBlock) RemoveParent(id uint) {
	if !b.HasParents() {
		return
	}
	b.parents.Remove(id)
}

// Get all parents set,the dag block has more than one parent
func (b *OldBlock) GetParents() *IdSet {
	return b.parents
}

func (b *OldBlock) GetMainParent() uint {
	return b.mainParent
}

// Testing whether it has parents
func (b *OldBlock) HasParents() bool {
	if b.parents == nil {
		return false
	}
	if b.parents.IsEmpty() {
		return false
	}
	return true
}

// Add child nodes to block
func (b *OldBlock) AddChild(child IBlock) {
	if b.children == nil {
		b.children = NewIdSet()
	}
	b.children.AddPair(child.GetID(), child)
}

// Get all the children of block
func (b *OldBlock) GetChildren() *IdSet {
	return b.children
}

// Detecting the presence of child nodes
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

// Setting the weight of block
func (b *OldBlock) SetWeight(weight uint64) {
	b.weight = weight
	if b.state != nil {
		b.state.SetWeight(b.weight)
	}
}

// Acquire the weight of blue blocks
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
	if b.state != nil {
		b.state.SetOrder(uint64(o))
	}
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
	// children
	children := []uint{}
	if b.HasChildren() {
		children = b.children.List()
	}
	childrenSize := len(children)
	err = s.WriteElements(w, uint32(childrenSize))
	if err != nil {
		return err
	}
	for i := 0; i < childrenSize; i++ {
		err = s.WriteElements(w, uint32(children[i]))
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
	// children
	var childrenSize uint32
	err = s.ReadElements(r, &childrenSize)
	if err != nil {
		return err
	}
	if childrenSize > 0 {
		b.children = NewIdSet()
		for i := uint32(0); i < childrenSize; i++ {
			var children uint32
			err := s.ReadElements(r, &children)
			if err != nil {
				return err
			}
			b.children.Add(uint(children))
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
	b.status = model.BlockStatus(status)
	return nil
}

// SetStatus
func (b *OldBlock) SetStatus(status model.BlockStatus) {
	b.status = status
}

func (b *OldBlock) GetStatus() model.BlockStatus {
	return b.status
}

func (b *OldBlock) SetStatusFlags(flags model.BlockStatus) {
	b.status |= flags
}

func (b *OldBlock) UnsetStatusFlags(flags model.BlockStatus) {
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
	b.UnsetStatusFlags(model.StatusInvalid)
	if b.state != nil {
		b.state.Valid()
	}
}

func (b *OldBlock) Invalid() {
	b.SetStatusFlags(model.StatusInvalid)
	if b.state != nil {
		b.state.Invalid()
	}
}

func (b *OldBlock) AttachParent(ib IBlock) {
	if ib == nil {
		return
	}
	if !b.HasParents() {
		return
	}
	if !b.parents.Has(ib.GetID()) {
		return
	}
	b.AddParent(ib)
}

func (b *OldBlock) DetachParent(ib IBlock) {
	if ib == nil {
		return
	}
	if !b.HasParents() {
		return
	}
	if !b.parents.Has(ib.GetID()) {
		return
	}
	b.parents.Add(ib.GetID())
}

func (b *OldBlock) AttachChild(ib IBlock) {
	if ib == nil {
		return
	}
	if !b.HasChildren() {
		return
	}
	if !b.children.Has(ib.GetID()) {
		return
	}
	b.AddChild(ib)
}

func (b *OldBlock) DetachChild(ib IBlock) {
	if ib == nil {
		return
	}
	if !b.HasChildren() {
		return
	}
	if !b.children.Has(ib.GetID()) {
		return
	}
	b.children.Add(ib.GetID())
}

func (b *OldBlock) Bytes() []byte {
	var buff bytes.Buffer
	err := b.Encode(&buff)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return buff.Bytes()
}

// GetState
func (b *OldBlock) GetState() model.BlockState {
	return b.state
}

func getOldBlockId(db legacydb.DB, h *hash.Hash) uint {
	if h == nil {
		return MaxId
	}
	id := MaxId
	err := db.View(func(dbTx legacydb.Tx) error {
		bid, er := DBGetBlockIdByHash(dbTx, h)
		if er == nil {
			id = uint(bid)
		}
		return er
	})
	if err != nil {
		log.Error(err.Error())
		return MaxId
	}
	return id
}

func getOldBlockById(db legacydb.DB, id uint) IBlock {
	block := OldBlock{id: id}
	ib := &OldPhantomBlock{&block, 0, nil, nil}
	err := db.View(func(dbTx legacydb.Tx) error {
		return DBGetDAGBlock(dbTx, ib)
	})
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	if id == 0 && !ib.GetHash().IsEqual(params.ActiveNetParams.GenesisHash) {
		log.Error("genesis data mismatch", "cur", ib.GetHash().String(), "genesis", params.ActiveNetParams.GenesisHash.String())
		return nil
	}
	return ib
}

func getOldBlock(db legacydb.DB, h *hash.Hash) IBlock {
	return getOldBlockById(db, getOldBlockId(db, h))
}

func getOldBlockByOrder(db legacydb.DB, order uint) IBlock {
	if order >= MaxBlockOrder {
		return nil
	}
	bid := uint(MaxId)
	err := db.View(func(dbTx legacydb.Tx) error {
		id, er := DBGetBlockIdByOrder(dbTx, order)
		if er == nil {
			bid = uint(id)
		}
		return er
	})
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return getOldBlockById(db, bid)
}

func makeConfigNode(cfg *config.Config) (*node.Node, *cli.Context, *eth.Config) {
	eth.InitLog(cfg.DebugLevel, cfg.DebugPrintOrigins)
	//
	var ecfg *eth.Config
	var args []string
	var err error

	ecfg, args, err = meer.MakeParams(cfg)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, nil
	}
	var n *node.Node
	var ctx *cli.Context
	n, ctx, err = eth.MakeNakedNode(ecfg, args)
	if err != nil {
		log.Error(err.Error())
		return nil, nil, nil
	}
	return n, ctx, ecfg
}

func getBlockNumber(db ethdb.Database, bh *hash.Hash) uint64 {
	bn := meer.ReadBlockNumber(db, qcommon.ToEVMHash(bh))
	if bn == nil {
		return 0
	}
	return *bn
}
