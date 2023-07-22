/*
 * Copyright (c) 2020.
 * Project:qitmeer
 * File:idset_test.go
 * Date:3/29/20 9:11 PM
 * Author:Jin
 * Email:lochjin@gmail.com
 */

package meerdag

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"testing"
	"time"
)

func Test_AddId(t *testing.T) {
	hs := NewIdSet()
	hs.Add(1)

	if !hs.Has(1) {
		t.FailNow()
	}
}

func Test_AddSetId(t *testing.T) {
	hs := NewIdSet()
	other := NewIdSet()
	other.Add(1)

	hs.AddSet(other)
	if !hs.Has(1) {
		t.FailNow()
	}
}

func Test_AddPairId(t *testing.T) {
	var intData int = 123
	hs := NewIdSet()
	hs.AddPair(1, int(intData))

	if !hs.Has(1) || hs.Get(1).(int) != intData {
		t.FailNow()
	}
}

func Test_RemoveId(t *testing.T) {
	hs := NewIdSet()
	hs.Add(1)
	hs.Remove(1)

	if hs.Has(1) {
		t.FailNow()
	}
}

func Test_RemoveSetId(t *testing.T) {
	hs := NewIdSet()
	other := NewIdSet()
	other.Add(1)

	hs.AddSet(other)
	hs.RemoveSet(other)

	if hs.Has(1) {
		t.FailNow()
	}
}

func Test_SortListId(t *testing.T) {
	hs := NewIdSet()
	hl := IdSlice{}
	var hashNum uint = 5
	for i := uint(0); i < hashNum; i++ {
		hs.Add(i)
		hl = append(hl, i)
	}
	shs := hs.SortList(false)

	for i := uint(0); i < hashNum; i++ {
		if hl[i] != shs[i] {
			t.FailNow()
		}
	}
	rshs := hs.SortList(true)

	for i := uint(0); i < hashNum; i++ {
		if hl[i] != rshs[hashNum-i-1] {
			t.FailNow()
		}
	}
}

func Test_SortListHash(t *testing.T) {
	hs := NewIdSet()
	hl := BlockHashSlice{}
	var hashNum uint = 5
	for i := uint(0); i < hashNum; i++ {
		hashStr := fmt.Sprintf("%d", i)
		h := hash.MustHexToDecodedHash(hashStr)
		block := &Block{id: i, hash: h}
		hs.AddPair(block.GetID(), block)
		hl = append(hl, block)
	}
	shs := hs.SortHashList(false)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != shs[i] {
			t.FailNow()
		}
	}
	rshs := hs.SortHashList(true)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != rshs[hashNum-i-1] {
			t.FailNow()
		}
	}
}

func Test_ForId(t *testing.T) {
	hs := NewIdSet()
	var hashNum uint = 5
	for i := uint(0); i < hashNum; i++ {
		hs.AddPair(i, i)
	}
	for k, v := range hs.GetMap() {
		fmt.Printf("%d - %d\n", v, k)
	}
}

// DAG block data
type TestBlock struct {
}

// Return the hash
func (tb *TestBlock) GetHash() *hash.Hash {
	return &hash.ZeroHash
}

// Get all parents set,the dag block has more than one parent
func (tb *TestBlock) GetParents() []*hash.Hash {
	return nil
}

func (tb *TestBlock) GetMainParent() *hash.Hash {
	return nil
}

func (tb *TestBlock) GetTimestamp() int64 {
	return time.Now().Unix()
}

// Acquire the weight of block
func (tb *TestBlock) GetWeight() uint64 {
	return 1
}

func (tb *TestBlock) GetPriority() int {
	return MaxPriority
}

func Test_SortListPriority(t *testing.T) {
	hs := NewIdSet()
	hl := BlockPrioritySlice{}
	var hashNum uint = 5
	for i := uint(0); i < hashNum; i++ {
		hashStr := fmt.Sprintf("%d", i)
		h := hash.MustHexToDecodedHash(hashStr)
		block := &PhantomBlock{Block: &Block{id: i, hash: h, data: &TestBlock{}}, blueNum: i}
		hs.AddPair(block.GetID(), block)
		hl = append(hl, block)
	}

	shs := hs.SortPriorityList(false)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != shs[i] {
			t.FailNow()
		}
	}
	rshs := hs.SortPriorityList(true)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != rshs[hashNum-i-1] {
			t.FailNow()
		}
	}
}

func Test_SortListHeight(t *testing.T) {
	hs := NewIdSet()
	hl := BlockHeightSlice{}
	var hashNum uint = 5
	for i := uint(0); i < hashNum; i++ {
		hashStr := fmt.Sprintf("%d", i)
		h := hash.MustHexToDecodedHash(hashStr)
		block := &Block{id: i, hash: h, height: i}
		hs.AddPair(block.GetID(), block)
		hl = append(hl, block)
	}
	shs := hs.SortHeightList(false)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != shs[i] {
			t.FailNow()
		}
	}
	rshs := hs.SortHeightList(true)

	for i := uint(0); i < hashNum; i++ {
		if hl[i].GetID() != rshs[hashNum-i-1] {
			t.FailNow()
		}
	}
}

func TestIsDataEmpty(t *testing.T) {
	hs := NewIdSet()
	hs.Add(1)
	hs.AddPair(2, int(2))

	if !hs.IsDataEmpty(1) {
		t.Fatalf("IsDataEmpty:%d = %v is not %v", 1, true, hs.IsDataEmpty(1))
	}
	if hs.IsDataEmpty(2) {
		t.Fatalf("IsDataEmpty:%d = %v is not %v", 1, false, hs.IsDataEmpty(1))
	}
}
