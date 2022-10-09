package meerdag

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	s "github.com/Qitmeer/qng/core/serialization"
	"github.com/Qitmeer/qng/database"
	"github.com/Qitmeer/qng/params"
	"io"
	"math"
	"time"
)

// Load from database
func (bd *MeerDAG) Load(dbTx database.Tx, blockTotal uint, genesis *hash.Hash) error {
	meta := dbTx.Metadata()
	serializedData := meta.Get(DagInfoBucketName)
	if serializedData == nil {
		return fmt.Errorf("dag load error")
	}

	err := bd.Decode(bytes.NewReader(serializedData))
	if err != nil {
		return err
	}
	bd.genesis = *genesis
	bd.blockTotal = blockTotal
	bd.blocks = map[uint]IBlock{}
	bd.tips = NewIdSet()
	return bd.instance.Load(dbTx)
}

func (bd *MeerDAG) Encode(w io.Writer) error {
	dagTypeIndex := GetDAGTypeIndex(bd.instance.GetName())
	err := s.WriteElements(w, dagTypeIndex)
	if err != nil {
		return err
	}
	return bd.instance.Encode(w)
}

// decode
func (bd *MeerDAG) Decode(r io.Reader) error {
	var dagTypeIndex byte
	err := s.ReadElements(r, &dagTypeIndex)
	if err != nil {
		return err
	}
	if GetDAGTypeIndex(bd.instance.GetName()) != dagTypeIndex {
		return fmt.Errorf("The dag type is %s, but read is %s", bd.instance.GetName(), GetDAGTypeByIndex(dagTypeIndex))
	}
	return bd.instance.Decode(r)
}

func (bd *MeerDAG) GetBlockData(ib IBlock) IBlockData {
	bd.blockDataLock.Lock()
	defer bd.blockDataLock.Unlock()

	bd.blockDataCache[ib.GetID()] = time.Now()
	if ib.IsLoaded() {
		return ib.GetData()
	}
	// load
	fmt.Println("加载blockdata:", ib.GetHash().String())
	data := bd.getBlockData(ib.GetHash())
	if data == nil {
		panic(fmt.Errorf("Can't load block data:%s", ib.GetHash().String()))
	}
	ib.SetData(data)
	return ib.GetData()
}

func (bd *MeerDAG) updateBlockDataCache() {
	bd.blockDataLock.Lock()
	defer bd.blockDataLock.Unlock()

	if len(bd.blockDataCache) <= MinBlockDataCache {
		return
	}
	maxLife := params.ActiveNetParams.TargetTimePerBlock
	startTime := time.Now()
	mainHeight := bd.GetMainChainTip().GetHeight()
	for k, t := range bd.blockDataCache {
		if time.Since(t) > maxLife {
			ib := bd.GetBlockById(k)
			if ib != nil {
				if math.Abs(float64(mainHeight)-float64(ib.GetHeight())) <= float64(MinBlockDataCache) {
					continue
				}
				ib.SetData(nil)
			}
			delete(bd.blockDataCache, k)
			fmt.Println("释放blockdata:", k)
		}
		if time.Since(startTime) > time.Second*2 {
			break
		}
	}
}

func (bd *MeerDAG) LoadBlockDataSet(sets *IdSet) {
	if sets == nil {
		return
	}
	for _, v := range sets.GetMap() {
		bd.GetBlockData(v.(IBlock))
	}
}

func (bd *MeerDAG) GetBlockDataCacheSize() int {
	bd.blockDataLock.Lock()
	defer bd.blockDataLock.Unlock()
	return len(bd.blockDataCache)
}