package miner

import (
	ejson "encoding/json"
	"fmt"
	"time"

	"github.com/Qitmeer/qng/services/mining"
)

const STATS_SUBMIT_100_AVG_DURATION = 100

type DURATIONLIST []time.Duration

// mining stats
type MiningStats struct {
	LastestGbt                        time.Time    `json:"lastest_gbt"`
	LastestGbtRequest                 time.Time    `json:"lastest_gbt_request"`
	LastestSubmit                     time.Time    `json:"lastest_submit"`
	Lastest100GbtRequests             []int64      `json:"-"`
	Lastest100Gbts                    []int64      `json:"-"`
	Lastest100GbtAvgDuration          float64      `json:"lastest_100_gbt_avg_duration"`
	Lastest100GbtRequestAvgDuration   float64      `json:"lastest_100_gbt_request_avg_duration"`
	Last100Submits                    DURATIONLIST `json:"-"`
	SubmitAvgDuration                 float64      `json:"submit_avg_duration"`
	TotalSubmitDuration               float64      `json:"total_submit_duration"`
	TotalGbtDuration                  float64      `json:"total_gbt_duration"`
	GbtAvgDuration                    float64      `json:"gbt_avg_duration"`
	TotalGbtRequestDuration           float64      `json:"total_gbt_request_duration"`
	GbtRequestAvgDuration             float64      `json:"gbt_request_avg_duration"`
	MaxGbtDuration                    float64      `json:"max_gbt_duration"`
	MaxGbtRequestDuration             float64      `json:"max_gbt_request_duration"`
	MaxGbtRequestTimeLongpollid       string       `json:"max_gbt_time_longpollid"`
	MaxSubmitDuration                 float64      `json:"max_submit_duration"`
	MaxSubmitDurationBlockHash        string       `json:"max_submit_duration_block_hash"`
	TotalGbts                         int64        `json:"total_gbts"`
	TotalGbtRequests                  int64        `json:"total_gbt_requests"`
	TotalEmptyGbts                    int64        `json:"total_empty_gbts"`
	TotalEmptyGbtDuarations           int64        `json:"total_empty_gbt_duarations"`
	TotalEmptyGbtResponse             int64        `json:"total_empty_gbt_response"`
	TotalSubmits                      int64        `json:"total_submits"`
	TotalTxEmptySubmits               int64        `json:"total_tx_empty_submits"`
	LastestMempoolEmptyTimestamp      int64        `json:"-"`
	TotalMempoolEmptyDuration         float64      `json:"total_mempool_empty_duration"`
	MempoolEmptyAvgDuration           float64      `json:"mempool_empty_avg_duration"`
	MempoolEmptyMaxDuration           float64      `json:"mempool_empty_max_duration"`
	MempoolEmptyWarns                 int64        `json:"mempool_empty_warns"`
	Lastest100MempoolEmptyDuration    []float64    `json:"lastest_100_mempool_empty_duration"`
	Lastest100MempoolEmptyAvgDuration float64      `json:"lastest_100_mempool_empty_avg_duration"`
}

func (ms *MiningStats) MarshalJSON() ([]byte, error) {
	tFormat := "2006-01-02 15:04:05"
	type MiningStatsOutput MiningStats
	tmpMiningStats := struct {
		MiningStatsOutput
		LastestGbt                        string `json:"lastest_gbt"`
		LastestGbtRequest                 string `json:"lastest_gbt_request"`
		LastestSubmit                     string `json:"lastest_submit"`
		Lastest1MempoolEmptyDuration      string `json:"lastest_1_mempool_empty_duration"`
		Lastest100GbtAvgDuration          string `json:"lastest_100_gbt_avg_duration"`
		Lastest100GbtRequestAvgDuration   string `json:"lastest_100_gbt_request_avg_duration"`
		Last100SubmitAvgDuration          string `json:"last_100_submit_avg_duration"`
		SubmitAvgDuration                 string `json:"submit_avg_duration"`
		GbtAvgDuration                    string `json:"gbt_avg_duration"`
		GbtRequestAvgDuration             string `json:"gbt_request_avg_duration"`
		MaxGbtDuration                    string `json:"max_gbt_duration"`
		MaxGbtRequestDuration             string `json:"max_gbt_request_duration"`
		MaxSubmitDuration                 string `json:"max_submit_duration"`
		MempoolEmptyAvgDuration           string `json:"mempool_empty_avg_duration"`
		Lastest100MempoolEmptyAvgDuration string `json:"lastest_100_mempool_empty_avg_duration"`
		MempoolEmptyMaxDuration           string `json:"mempool_empty_max_duration"`
	}{
		MiningStatsOutput:                 (MiningStatsOutput)(*ms),
		LastestGbt:                        ms.LastestGbt.Format(tFormat),
		LastestGbtRequest:                 ms.LastestGbtRequest.Format(tFormat),
		LastestSubmit:                     ms.LastestSubmit.Format(tFormat),
		Lastest100GbtAvgDuration:          fmt.Sprintf("%.3f s", ms.Lastest100GbtAvgDuration),
		Lastest100GbtRequestAvgDuration:   fmt.Sprintf("%.3f s", ms.Lastest100GbtRequestAvgDuration),
		SubmitAvgDuration:                 fmt.Sprintf("%.3f s", ms.SubmitAvgDuration),
		GbtAvgDuration:                    fmt.Sprintf("%.3f s", ms.GbtAvgDuration),
		GbtRequestAvgDuration:             fmt.Sprintf("%.3f s", ms.GbtRequestAvgDuration),
		MaxGbtDuration:                    fmt.Sprintf("%.3f s", ms.MaxGbtDuration),
		MaxGbtRequestDuration:             fmt.Sprintf("%.3f s", ms.MaxGbtRequestDuration),
		MaxSubmitDuration:                 fmt.Sprintf("%.3f s", ms.MaxSubmitDuration),
		MempoolEmptyAvgDuration:           fmt.Sprintf("%.3f s", ms.MempoolEmptyAvgDuration),
		Lastest100MempoolEmptyAvgDuration: fmt.Sprintf("%.3f s", ms.Lastest100MempoolEmptyAvgDuration),
		MempoolEmptyMaxDuration:           fmt.Sprintf("%.3f s", ms.MempoolEmptyMaxDuration),
	}
	if len(ms.Lastest100MempoolEmptyDuration) > 0 {
		tmpMiningStats.Lastest1MempoolEmptyDuration = fmt.Sprintf("%.3f s",
			ms.Lastest100MempoolEmptyDuration[len(ms.Lastest100MempoolEmptyDuration)-1 : len(ms.Lastest100MempoolEmptyDuration)][0])
	}
	return ejson.Marshal(tmpMiningStats)
}

func (dl *DURATIONLIST) CalcLast100SubmitAvgDuration() time.Duration {
	if dl.Length() < 1 {
		return mining.TargetAllowSubmitHandleDuration
	}
	allDuration := time.Duration(0)
	for _, v := range *dl {
		allDuration += v
	}
	return allDuration / time.Duration(len(*dl))
}

func (dl *DURATIONLIST) Length() int {
	if dl == nil {
		return 0
	}
	return len(*dl)
}

func (m *Miner) StatsEmptyGbt() {
	if m.stats.LastestMempoolEmptyTimestamp <= 0 {
		m.stats.LastestMempoolEmptyTimestamp = time.Now().Unix()
	}
}

func (m *Miner) StatsGbtTxEmptyAvgTimes() {
	if m.stats.LastestMempoolEmptyTimestamp <= 0 || time.Now().Unix() <= m.stats.LastestMempoolEmptyTimestamp {
		return
	}
	m.stats.TotalEmptyGbtDuarations++
	duration := float64(time.Now().Unix() - m.stats.LastestMempoolEmptyTimestamp)
	m.stats.TotalMempoolEmptyDuration += duration
	mempoolEmptyDuration.Update(time.Duration(duration) * time.Second)
	if m.stats.TotalEmptyGbtDuarations > 0 {
		m.stats.MempoolEmptyAvgDuration = m.stats.TotalMempoolEmptyDuration / float64(m.stats.TotalEmptyGbtDuarations)
	}

	if len(m.stats.Lastest100MempoolEmptyDuration) >= 100 {
		m.stats.Lastest100MempoolEmptyDuration = m.stats.Lastest100MempoolEmptyDuration[len(m.stats.Lastest100MempoolEmptyDuration)-99:]
	}
	m.stats.Lastest100MempoolEmptyDuration = append(m.stats.Lastest100MempoolEmptyDuration, duration)
	sum := float64(0)
	for _, v := range m.stats.Lastest100MempoolEmptyDuration {
		sum += v
	}
	m.stats.Lastest100MempoolEmptyAvgDuration = float64(sum) / float64(len(m.stats.Lastest100MempoolEmptyDuration))

	if duration > m.stats.MempoolEmptyMaxDuration {
		m.stats.MempoolEmptyMaxDuration = duration
	}
}

func (m *Miner) StatsSubmit(start time.Time, bh string, txcount int) {
	currentReqMillSec := time.Since(start).Milliseconds()
	m.stats.TotalSubmits++
	totalSubmits.Update(m.stats.TotalSubmits)
	if m.stats.Last100Submits.Length() >= STATS_SUBMIT_100_AVG_DURATION {
		m.stats.Last100Submits = m.stats.Last100Submits[len(m.stats.Last100Submits)-STATS_SUBMIT_100_AVG_DURATION+1:]
	}
	m.stats.Last100Submits = append(m.stats.Last100Submits, time.Since(start))

	m.stats.TotalSubmitDuration += float64(currentReqMillSec) / 1000
	if m.stats.TotalSubmits > 0 {
		m.stats.SubmitAvgDuration = m.stats.TotalSubmitDuration / float64(m.stats.TotalSubmits)
		if m.cfg.UseDynamicBlockMaxSize && m.stats.TotalSubmits%STATS_SUBMIT_100_AVG_DURATION == 0 {
			// stats once every 100 submits
			m.policy.CalcMaxBlockSize(m.stats.Last100Submits.CalcLast100SubmitAvgDuration())
		}
	}
	submitDuration.Update(time.Since(start))
	if float64(currentReqMillSec)/1000 > m.stats.MaxSubmitDuration {
		m.stats.MaxSubmitDuration = float64(currentReqMillSec) / 1000
		m.stats.MaxSubmitDurationBlockHash = bh
	}
	submitTxCount.Update(int64(txcount))
	if txcount < 1 {
		m.stats.TotalTxEmptySubmits++
		totalTxEmptySubmits.Update(m.stats.TotalTxEmptySubmits)
	}
}

func (m *Miner) StatsGbtRequest(currentReqMillSec int64, txcount int, longpollid string) {
	if len(m.stats.Lastest100GbtRequests) >= 100 {
		m.stats.Lastest100GbtRequests = m.stats.Lastest100GbtRequests[len(m.stats.Lastest100GbtRequests)-99:]
	}
	m.stats.Lastest100GbtRequests = append(m.stats.Lastest100GbtRequests, currentReqMillSec)
	sum := int64(0)
	for _, v := range m.stats.Lastest100GbtRequests {
		sum += v
	}
	m.stats.LastestGbtRequest = time.Now()
	m.stats.Lastest100GbtRequestAvgDuration = float64(sum) / float64(len(m.stats.Lastest100GbtRequests)) / 1000
	m.stats.TotalGbtRequestDuration += float64(currentReqMillSec) / 1000
	if m.stats.TotalGbtRequests > 0 {
		m.stats.GbtRequestAvgDuration = m.stats.TotalGbtRequestDuration / float64(m.stats.TotalGbtRequests)
	}
	gbtRequestDuration.Update(time.Duration(float64(currentReqMillSec)/1000) * time.Second)
	if float64(currentReqMillSec)/1000 > m.stats.MaxGbtRequestDuration {
		m.stats.MaxGbtRequestDuration = float64(currentReqMillSec) / 1000
		m.stats.MaxGbtRequestTimeLongpollid = longpollid
	}
	if txcount < 1 {
		m.stats.TotalEmptyGbtResponse++
	}
}

func (m *Miner) StatsGbt(currentReqMillSec int64, txcount int) {
	if len(m.stats.Lastest100Gbts) >= 100 {
		m.stats.Lastest100Gbts = m.stats.Lastest100Gbts[len(m.stats.Lastest100Gbts)-99:]
	}
	m.stats.Lastest100Gbts = append(m.stats.Lastest100Gbts, currentReqMillSec)
	sum := int64(0)
	for _, v := range m.stats.Lastest100Gbts {
		sum += v
	}
	m.stats.LastestGbt = time.Now()
	m.stats.Lastest100GbtAvgDuration = float64(sum) / float64(len(m.stats.Lastest100Gbts)) / 1000

	m.stats.TotalGbtDuration += float64(currentReqMillSec) / 1000
	if m.stats.TotalGbts > 0 {
		m.stats.GbtAvgDuration = m.stats.TotalGbtDuration / float64(m.stats.TotalGbts)
	}
	gbtDuration.Update(time.Duration(float64(currentReqMillSec)/1000) * time.Second)
	if float64(currentReqMillSec)/1000 > m.stats.MaxGbtDuration {
		m.stats.MaxGbtDuration = float64(currentReqMillSec) / 1000
	}
	if txcount < 1 {
		m.StatsEmptyGbt()
		m.stats.TotalEmptyGbts++
	} else {
		m.StatsGbtTxEmptyAvgTimes()
		m.stats.LastestMempoolEmptyTimestamp = 0
	}
}
