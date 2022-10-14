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
	data := bd.getBlockData(ib.GetHash())
	if data == nil {
		panic(fmt.Errorf("Can't load block data:%s", ib.GetHash().String()))
	}
	ib.SetData(data)
	return ib.GetData()
}

func (bd *MeerDAG) updateBlockDataCache() {
	cacheSize := bd.GetBlockDataCacheSize()
	if cacheSize <= MinBlockDataCache {
		return
	}
	maxLife := params.ActiveNetParams.TargetTimePerBlock
	startTime := time.Now()
	mainHeight := bd.GetMainChainTip().GetHeight()
	need := cacheSize - MinBlockDataCache
	blockDataCache := map[uint]time.Time{}
	bd.blockDataLock.Lock()
	const waitTime = time.Second * 2
	for k, t := range bd.blockDataCache {
		blockDataCache[k] = t
		need--
		if need <= 0 {
			break
		}
		if time.Since(startTime) > waitTime {
			break
		}
	}
	bd.blockDataLock.Unlock()
	for k, t := range blockDataCache {
		if time.Since(t) > maxLife {
			ib := bd.GetBlockById(k)
			if ib != nil {
				if math.Abs(float64(mainHeight)-float64(ib.GetHeight())) <= float64(MinBlockDataCache) {
					continue
				}
				ib.SetData(nil)
			}
			bd.blockDataLock.Lock()
			delete(bd.blockDataCache, k)
			bd.blockDataLock.Unlock()
		}
		if time.Since(startTime) > waitTime {
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
