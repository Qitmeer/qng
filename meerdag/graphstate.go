package meerdag

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
	"sync"
)

// A general description of the whole state of DAG
type GraphState struct {
	// The terminal block is in block dag,this block have not any connecting at present.
	tips    []hash.Hash
	tipsSet *HashSet

	// The total number blocks that this dag currently owned
	total uint

	// At present, the whole graph nodes has the last layer level.
	layer uint

	// The height of main chain
	mainHeight uint

	// The order of main chain tip
	mainOrder uint

	sync.RWMutex
}

// Return the DAG layer
func (gs *GraphState) GetLayer() uint {
	return gs.layer
}

func (gs *GraphState) SetLayer(layer uint) {
	gs.layer = layer
}

// Return the total of DAG
func (gs *GraphState) GetTotal() uint {
	return gs.total
}

func (gs *GraphState) SetTotal(total uint) {
	gs.total = total
}

func (gs *GraphState) GetMainOrder() uint {
	return gs.mainOrder
}

func (gs *GraphState) SetMainOrder(order uint) {
	gs.mainOrder = order
}

// Return all tips of DAG
func (gs *GraphState) GetTipsList() []*hash.Hash {
	ret := []*hash.Hash{}
	for _, h := range gs.tips {
		ha := h
		ret = append(ret, &ha)
	}
	return ret
}

func (gs *GraphState) SetTips(tips []*hash.Hash) {
	gs.Lock()
	defer gs.Unlock()

	gs.tipsSet = nil
	gs.tips = []hash.Hash{}
	for _, h := range tips {
		gs.tips = append(gs.tips, *h)
	}
}

func (gs *GraphState) GetTips() *HashSet {
	gs.Lock()
	defer gs.Unlock()

	if gs.tipsSet != nil {
		return gs.tipsSet
	}
	gs.tipsSet = NewHashSet()
	for k, v := range gs.tips {
		h := v
		if k == 0 {
			gs.tipsSet.AddPair(&h, true)
		} else {
			gs.tipsSet.Add(&h)
		}
	}
	return gs.tipsSet
}

// Return the height of main chain
func (gs *GraphState) GetMainHeight() uint {
	return gs.mainHeight
}

func (gs *GraphState) SetMainHeight(mainHeight uint) {
	gs.mainHeight = mainHeight
}

// Judging whether it is equal to other
func (gs *GraphState) IsEqual(other *GraphState) bool {
	if gs == other {
		return true
	}
	if gs.layer != other.layer ||
		gs.mainOrder != other.mainOrder ||
		gs.mainHeight != other.mainHeight {
		return false
	}
	tipsSet := gs.GetTips()
	otherTipsSet := other.GetTips()

	if tipsSet.Contain(otherTipsSet) ||
		otherTipsSet.Contain(tipsSet) {
		return true
	}
	return false
}

// Setting vaules from other
func (gs *GraphState) Equal(other *GraphState) {
	if gs.IsEqual(other) {
		return
	}
	gs.SetTips(other.GetTipsList())
	gs.layer = other.layer
	gs.total = other.total
	gs.mainHeight = other.mainHeight
	gs.mainOrder = other.mainOrder
}

// Copy self and return
func (gs *GraphState) Clone() *GraphState {
	result := NewGraphState()
	result.Equal(gs)
	return result
}

// Return one string contain info
func (gs *GraphState) String() string {
	return fmt.Sprintf("(%d,%d,%d,%d,%d)", gs.mainOrder, gs.mainHeight, gs.layer, gs.total, len(gs.tips))
}

// Judging whether it is better than other
func (gs *GraphState) IsExcellent(other *GraphState) bool {
	if gs.IsEqual(other) {
		return false
	}
	if gs.mainOrder < other.mainOrder {
		return false
	} else if gs.mainOrder > other.mainOrder {
		return true
	}
	if gs.mainHeight < other.mainHeight {
		return false
	} else if gs.mainHeight > other.mainHeight {
		return true
	}
	if gs.layer < other.layer {
		return false
	} else if gs.layer > other.layer {
		return true
	}
	if gs.GetMainChainTip() != nil &&
		other.GetMainChainTip() != nil {
		if gs.GetMainChainTip().String() < other.GetMainChainTip().String() {
			return true
		}
	}
	return false
}

// Encode itself to bytes buff
func (gs *GraphState) Encode(w io.Writer, pver uint32) error {
	err := s.WriteVarInt(w, pver, uint64(gs.total))
	if err != nil {
		return err
	}
	err = s.WriteVarInt(w, pver, uint64(gs.layer))
	if err != nil {
		return err
	}
	err = s.WriteVarInt(w, pver, uint64(gs.mainHeight))
	if err != nil {
		return err
	}
	err = s.WriteVarInt(w, pver, uint64(gs.mainOrder))
	if err != nil {
		return err
	}
	err = s.WriteVarInt(w, pver, uint64(len(gs.tips)))
	if err != nil {
		return err
	}

	for _, v := range gs.tips {
		err = s.WriteElements(w, &v)
		if err != nil {
			return err
		}
	}

	return nil
}

// Decode itself from bytes buff
func (gs *GraphState) Decode(r io.Reader, pver uint32) error {
	total, err := s.ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	gs.total = uint(total)

	layer, err := s.ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	gs.layer = uint(layer)

	mainHeight, err := s.ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	gs.mainHeight = uint(mainHeight)

	mainOrder, err := s.ReadVarInt(r, pver)
	if err != nil {
		return err
	}
	gs.mainOrder = uint(mainOrder)

	count, err := s.ReadVarInt(r, pver)
	if count == 0 || err != nil {
		return fmt.Errorf("GraphState.Decode:tips count is zero.:%v", err)
	}

	locatorHashes := []*hash.Hash{}
	for i := uint64(0); i < count; i++ {
		var h hash.Hash
		err := s.ReadElements(r, &h)
		if err != nil {
			return err
		}
		locatorHashes = append(locatorHashes, &h)
	}
	gs.SetTips(locatorHashes)
	return nil
}

func (gs *GraphState) MaxPayloadLength() uint32 {
	return 8 + 4 + 4 + 4 + (MaxTips * hash.HashSize)
}

func (gs *GraphState) GetMainChainTip() *hash.Hash {
	if len(gs.tips) > 0 {
		return &gs.tips[0]
	}
	return nil
}

func (gs *GraphState) IsGenesis() bool {
	if len(gs.tips) == 1 &&
		gs.total == 1 &&
		gs.mainHeight == 0 &&
		gs.mainOrder == 0 &&
		gs.layer == 0 {
		return true
	}
	return false
}

// Create a new GraphState
func NewGraphState() *GraphState {
	return &GraphState{
		tips:       []hash.Hash{},
		total:      0,
		layer:      0,
		mainHeight: 0,
		mainOrder:  0,
	}
}
