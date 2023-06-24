package meerdag

import (
	"bytes"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

// The abstract inferface is used to dag block
type IBlockData interface {
	// Get hash of block
	GetHash() *hash.Hash

	// Get all parents set,the dag block has more than one parent
	GetParents() []*hash.Hash

	GetMainParent() *hash.Hash

	// Timestamp
	GetTimestamp() int64

	// Priority
	GetPriority() int

	Clean()
}

// The interface of block
type IBlock interface {
	// Return block ID
	GetID() uint

	// Return the hash of block. It will be a pointer.
	GetHash() *hash.Hash

	AddParent(parent IBlock)

	// Get all parents set,the dag block has more than one parent
	GetParents() *IdSet

	// Testing whether it has parents
	HasParents() bool

	// Add child nodes to block
	AddChild(child IBlock)

	// Get all the children of block
	GetChildren() *IdSet

	// Detecting the presence of child nodes
	HasChildren() bool

	RemoveChild(child uint)

	// GetMainParent
	GetMainParent() uint

	// Acquire the layer of block
	GetLayer() uint

	// Acquire the height of block in main chain
	GetHeight() uint

	// encode
	Encode(w io.Writer) error

	// decode
	Decode(r io.Reader) error

	// block data
	GetData() IBlockData
	SetData(data IBlockData)
	IsLoaded() bool

	AttachParent(ib IBlock)
	DetachParent(ib IBlock)

	AttachChild(ib IBlock)
	DetachChild(ib IBlock)

	// GetState
	GetState() model.BlockState

	// TODO:
	Bytes() []byte

	// Setting the order of block
	SetOrder(o uint)

	// Acquire the order of block
	GetOrder() uint

	// IsOrdered
	IsOrdered() bool
}

// It is the element of a DAG. It is the most basic data unit.
type Block struct {
	id       uint
	hash     hash.Hash
	parents  *IdSet
	children *IdSet

	mainParent uint
	layer      uint
	height     uint

	data  IBlockData
	state model.BlockState
}

// Return block ID
func (b *Block) GetID() uint {
	return b.id
}

func (b *Block) SetID(id uint) {
	b.id = id
}

// Return the hash of block. It will be a pointer.
func (b *Block) GetHash() *hash.Hash {
	return &b.hash
}

func (b *Block) AddParent(parent IBlock) {
	if b.parents == nil {
		b.parents = NewIdSet()
	}
	b.parents.AddPair(parent.GetID(), parent)
}

func (b *Block) RemoveParent(id uint) {
	if !b.HasParents() {
		return
	}
	b.parents.Remove(id)
}

// Get all parents set,the dag block has more than one parent
func (b *Block) GetParents() *IdSet {
	return b.parents
}

func (b *Block) GetMainParent() uint {
	return b.mainParent
}

// Testing whether it has parents
func (b *Block) HasParents() bool {
	if b.parents == nil {
		return false
	}
	if b.parents.IsEmpty() {
		return false
	}
	return true
}

// Add child nodes to block
func (b *Block) AddChild(child IBlock) {
	if b.children == nil {
		b.children = NewIdSet()
	}
	b.children.AddPair(child.GetID(), child)
}

// Get all the children of block
func (b *Block) GetChildren() *IdSet {
	return b.children
}

// Detecting the presence of child nodes
func (b *Block) HasChildren() bool {
	if b.children == nil {
		return false
	}
	if b.children.IsEmpty() {
		return false
	}
	return true
}

func (b *Block) RemoveChild(child uint) {
	if !b.HasChildren() {
		return
	}
	b.children.Remove(child)
}

// Setting the layer of block
func (b *Block) SetLayer(layer uint) {
	b.layer = layer
}

// Acquire the layer of block
func (b *Block) GetLayer() uint {
	return b.layer
}

// Setting the order of block
func (b *Block) SetOrder(o uint) {
	b.state.SetOrder(uint64(o))
}

// Acquire the order of block
func (b *Block) GetOrder() uint {
	return uint(b.state.GetOrder())
}

// IsOrdered
func (b *Block) IsOrdered() bool {
	return b.state.IsOrdered()
}

// Setting the height of block in main chain
func (b *Block) SetHeight(h uint) {
	b.height = h
}

// Acquire the height of block in main chain
func (b *Block) GetHeight() uint {
	return b.height
}

// encode
func (b *Block) Encode(w io.Writer) error {
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
	err = s.WriteElements(w, uint32(b.layer))
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint32(b.height))
	if err != nil {
		return err
	}

	bs, err := b.state.Bytes()
	if err != nil {
		return err
	}
	err = s.WriteElements(w, uint64(len(bs)))
	if err != nil {
		return err
	}
	_, err = w.Write(bs)
	return err
}

// decode
func (b *Block) Decode(r io.Reader) error {
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

	//
	var bsSize uint64
	err = s.ReadElements(r, &bsSize)
	if err != nil {
		return err
	}
	bs := make([]byte, bsSize)
	_, err = r.Read(bs)
	if err != nil {
		return err
	}
	b.state, err = createBlockStateFromBytes(bs)
	if err != nil {
		return err
	}
	return nil
}

func (b *Block) GetData() IBlockData {
	return b.data
}

func (b *Block) SetData(data IBlockData) {
	b.data = data
}

func (b *Block) IsLoaded() bool {
	return b.data != nil
}

func (b *Block) AttachParent(ib IBlock) {
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

func (b *Block) DetachParent(ib IBlock) {
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

func (b *Block) AttachChild(ib IBlock) {
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

func (b *Block) DetachChild(ib IBlock) {
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

func (b *Block) Bytes() []byte {
	var buff bytes.Buffer
	err := b.Encode(&buff)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return buff.Bytes()
}

// GetState
func (b *Block) GetState() model.BlockState {
	return b.state
}
