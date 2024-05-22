// Copyright (c) 2019 The qitmeer developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
package lib

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Qitmeer/qng/cmd/miner/common"
	"github.com/Qitmeer/qng/cmd/miner/core"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/rpc/client"
)

var ErrSameWork = fmt.Errorf("Same work, Had Submitted!")
var ErrLimitWork = fmt.Errorf("Submission interval Limited")

type QitmeerWork struct {
	core.Work
	Block       *BlockHeader
	PoolWork    NotifyWork
	stra        *QitmeerStratum
	StartWork   bool
	ForceUpdate bool
	Ing         bool
	WorkLock    sync.Mutex
	WsClient    *client.Client
	LastSubmit  time.Time
	GbtID       int64
	SubmitID    int64
}

func (this *QitmeerWork) GetPowType() pow.PowType {
	switch this.Cfg.NecessaryConfig.Pow {
	case POW_MEER_CRYPTO:
		return pow.MEERXKECCAKV1
	default:
		return pow.BLAKE2BD
	}
}

// GetBlockTemplate
func (this *QitmeerWork) Get() bool {
	if this.Ing {
		return false
	}
	this.Ing = true
	if time.Since(this.LastSubmit) < time.Duration(this.Cfg.OptionConfig.TaskInterval)*time.Millisecond {
		<-time.After(time.Since(this.LastSubmit))
		this.Ing = false
		return this.Get()
	}

	defer func() {
		this.Ing = false
	}()
	for {
		if this.WsClient == nil || this.WsClient.Disconnected() {
			return false
		}
		this.ForceUpdate = false
		header, err := this.WsClient.GetRemoteGBT(this.GetPowType())
		if err != nil {
			time.Sleep(time.Duration(this.Cfg.OptionConfig.TaskInterval) * time.Millisecond)
			common.MinerLoger.Error("GetRemoteGBT Error", "err", err.Error())
			continue
		}
		if this.Block != nil && this.Block.ParentRoot == header.ParentRoot &&
			(time.Now().Unix()-this.GetWorkTime) < int64(this.Cfg.OptionConfig.Timeout)*10 {
			common.MinerLoger.Warn("GetRemoteGBT Repeat", "old block parent root", this.Block.ParentRoot, "current", header.ParentRoot)
			//not has new work
			return false
		}
		return this.BuildBlock(header)
	}
}

// BuildBlock
func (this *QitmeerWork) BuildBlock(header *types.BlockHeader) bool {
	this.GbtID++
	this.Block = &BlockHeader{}
	this.Block.ParentRoot = header.ParentRoot
	this.Block.WorkData = header.BlockData()
	this.Block.Target = fmt.Sprintf("%064x", pow.CompactToBig(header.Difficulty))
	this.Block.GBTID = this.GbtID
	common.LatestGBTID = this.GbtID
	common.MinerLoger.Debug(fmt.Sprintf("getRemoteBlockTemplate , target :%s , GBTID:%d", this.Block.Target, this.GbtID))
	this.GetWorkTime = time.Now().Unix()
	return true
}

// submit
func (this *QitmeerWork) Submit(header *types.BlockHeader, gbtID string) (string, int, error) {
	this.Lock()
	defer this.Unlock()
	gbtIDInt64, _ := strconv.ParseInt(gbtID, 10, 64)
	if this.GbtID != gbtIDInt64 {
		common.MinerLoger.Debug(fmt.Sprintf("gbt old , target :%d , current:%d", this.GbtID, gbtIDInt64))
		return "", 0, ErrSameWork
	}
	this.SubmitID++

	id := fmt.Sprintf("miner_submit_gbtID:%s_id:%d", gbtID, this.SubmitID)
	res, err := this.WsClient.SubmitBlockHeader(header)
	if err != nil {
		common.MinerLoger.Error("[submit error] " + id + " " + err.Error())
		if strings.Contains(err.Error(), "The tips of block is expired") {
			return "", 0, ErrSameWork
		}
		if strings.Contains(err.Error(), "worthless") {
			return "", 0, ErrSameWork
		}
		if strings.Contains(err.Error(), "Submission interval Limited") {
			return "", 0, ErrLimitWork
		}
		return "", 0, errors.New("[submit data failed]" + err.Error())
	}
	return res.CoinbaseTxID, int(res.Height), err
}

// pool get work
func (this *QitmeerWork) PoolGet() bool {
	if !this.stra.PoolWork.NewWork {
		return false
	}
	err := this.stra.PoolWork.PrepWork()
	if err != nil {
		common.MinerLoger.Error(err.Error())
		return false
	}

	if (this.stra.PoolWork.JobID != "" && this.stra.PoolWork.Clean) || this.PoolWork.JobID != this.stra.PoolWork.JobID {
		this.stra.PoolWork.Clean = false
		this.Cfg.OptionConfig.Target = fmt.Sprintf("%064x", common.BlockBitsToTarget(this.stra.PoolWork.Nbits, 2))
		this.PoolWork = this.stra.PoolWork
		common.CurrentHeight = uint64(this.stra.PoolWork.Height)
		common.JobID = this.stra.PoolWork.JobID
		return true
	}

	return false
}

// pool submit work
func (this *QitmeerWork) PoolSubmit(subm string) error {
	if this.LastSub == subm {
		return ErrSameWork
	}
	this.LastSub = subm
	arr := strings.Split(subm, "-")
	data, err := hex.DecodeString(arr[0])
	if err != nil {
		return err
	}
	sub, err := this.stra.PrepSubmit(data, arr[1], arr[2])
	if err != nil {
		return err
	}
	m, err := json.Marshal(sub)
	if err != nil {
		return err
	}
	_, err = this.stra.Conn.Write(m)
	if err != nil {
		common.MinerLoger.Debug("[submit error][pool connect error]", "error", err)
		return err
	}
	_, err = this.stra.Conn.Write([]byte("\n"))
	if err != nil {
		common.MinerLoger.Debug(err.Error())
		return err
	}

	return nil
}
