package meerdag

import (
	"bytes"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

type PhantomBlock struct {
	*Block
	blueNum uint

	blueDiffAnticone *IdSet
	redDiffAnticone  *IdSet
}

func (pb *PhantomBlock) IsBluer(other *PhantomBlock) bool {
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
func (pb *PhantomBlock) Encode(w io.Writer) error {
	err := pb.Block.Encode(w)
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
func (pb *PhantomBlock) Decode(r io.Reader) error {
	err := pb.Block.Decode(r)
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
func (pb *PhantomBlock) GetBlueNum() uint {
	return pb.blueNum
}

func (pb *PhantomBlock) GetBlueDiffAnticone() *IdSet {
	return pb.blueDiffAnticone
}

func (pb *PhantomBlock) GetRedDiffAnticone() *IdSet {
	return pb.redDiffAnticone
}

func (pb *PhantomBlock) GetDiffAnticoneList(filterType FilterType) []uint {
	if pb.GetDiffAnticoneSize() <= 0 {
		return nil
	}
	list := []uint{}
	for i := 0; i < pb.GetDiffAnticoneSize(); i++ {
		list = append(list, MaxId)
	}
	if filterType == Blue ||
		filterType == All {
		if pb.GetBlueDiffAnticoneSize() > 0 {
			for k, v := range pb.GetBlueDiffAnticone().GetMap() {
				idx := v.(uint) - 1
				list[idx] = k
			}
		}
	}
	if filterType == Red ||
		filterType == All {
		if pb.GetRedDiffAnticoneSize() > 0 {
			for k, v := range pb.GetRedDiffAnticone().GetMap() {
				idx := v.(uint) - 1
				list[idx] = k
			}
		}
	}
	return list
}

func (pb *PhantomBlock) GetBlueDiffAnticoneSize() int {
	if pb.blueDiffAnticone == nil {
		return 0
	}
	return pb.blueDiffAnticone.Size()
}

func (pb *PhantomBlock) GetRedDiffAnticoneSize() int {
	if pb.redDiffAnticone == nil {
		return 0
	}
	return pb.redDiffAnticone.Size()
}

func (pb *PhantomBlock) GetDiffAnticoneSize() int {
	return pb.GetBlueDiffAnticoneSize() + pb.GetRedDiffAnticoneSize()
}

func (pb *PhantomBlock) AddBlueDiffAnticone(id uint) {
	if pb.blueDiffAnticone == nil {
		pb.blueDiffAnticone = NewIdSet()
	}
	pb.blueDiffAnticone.Add(id)
}

func (pb *PhantomBlock) AddRedDiffAnticone(id uint) {
	if pb.redDiffAnticone == nil {
		pb.redDiffAnticone = NewIdSet()
	}
	pb.redDiffAnticone.Add(id)
}

func (pb *PhantomBlock) AddPairBlueDiffAnticone(id uint, order uint) {
	if pb.blueDiffAnticone == nil {
		pb.blueDiffAnticone = NewIdSet()
	}
	pb.blueDiffAnticone.AddPair(id, order)
}

func (pb *PhantomBlock) AddPairRedDiffAnticone(id uint, order uint) {
	if pb.redDiffAnticone == nil {
		pb.redDiffAnticone = NewIdSet()
	}
	pb.redDiffAnticone.AddPair(id, order)
}

func (pb *PhantomBlock) HasBlueDiffAnticone(id uint) bool {
	if pb.blueDiffAnticone == nil {
		return false
	}
	return pb.blueDiffAnticone.Has(id)
}

func (pb *PhantomBlock) HasRedDiffAnticone(id uint) bool {
	if pb.redDiffAnticone == nil {
		return false
	}
	return pb.redDiffAnticone.Has(id)
}

func (pb *PhantomBlock) CleanDiffAnticone() {
	if pb.blueDiffAnticone != nil {
		pb.blueDiffAnticone.Clean()
	}
	if pb.redDiffAnticone != nil {
		pb.redDiffAnticone.Clean()
	}
}

func (pb *PhantomBlock) Bytes() []byte {
	var buff bytes.Buffer
	err := pb.Encode(&buff)
	if err != nil {
		log.Error(err.Error())
		return nil
	}
	return buff.Bytes()
}
