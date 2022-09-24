package miner

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/marshal"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/services/mining"
	"sync"
	"sync/atomic"
)

type RemoteWorker struct {
	started  int32
	shutdown int32

	miner *Miner
	sync.Mutex
}

func (w *RemoteWorker) GetType() string {
	return RemoteWorkerType
}

func (w *RemoteWorker) Start() error {
	err := w.miner.initCoinbase()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	// Already started?
	if atomic.AddInt32(&w.started, 1) != 1 {
		return nil
	}

	log.Info("Start Remote Worker...")
	w.miner.updateBlockTemplate(false, false)
	return nil
}

func (w *RemoteWorker) Stop() {
	if atomic.AddInt32(&w.shutdown, 1) != 1 {
		log.Warn(fmt.Sprintf("Remote Worker is already in the process of shutting down"))
		return
	}
	log.Info("Stop Remote Worker...")

}

func (w *RemoteWorker) IsRunning() bool {
	return atomic.LoadInt32(&w.started) != 0
}

func (w *RemoteWorker) Update() {
	if atomic.LoadInt32(&w.shutdown) != 0 {
		return
	}
}

func (w *RemoteWorker) GetRequest(powType pow.PowType, coinbaseFlags mining.CoinbaseFlags, reply chan *gbtResponse) {
	if atomic.LoadInt32(&w.shutdown) != 0 {
		reply <- &gbtResponse{nil, fmt.Errorf("RemoteWorker is not running ")}
		return
	}

	w.Lock()
	defer w.Unlock()

	if w.miner.powType != powType {
		log.Info(fmt.Sprintf("%s:Change pow type %s => %s", w.GetType(), pow.GetPowName(w.miner.powType), pow.GetPowName(powType)))
		w.miner.powType = powType
		if err := w.miner.updateBlockTemplate(true, false); err != nil {
			reply <- &gbtResponse{nil, err}
			return
		}
	}
	if w.miner.coinbaseFlags != coinbaseFlags {
		log.Info(fmt.Sprintf("%s:Change coinbase flags %s => %s", w.GetType(), w.miner.coinbaseFlags, coinbaseFlags))
		w.miner.coinbaseFlags = coinbaseFlags
		if err := w.miner.updateBlockTemplate(true, false); err != nil {
			reply <- &gbtResponse{nil, err}
			return
		}
	}
	var headerBuf bytes.Buffer
	err := w.miner.template.Block.Header.Serialize(&headerBuf)
	if err != nil {
		reply <- &gbtResponse{nil, err}
		return
	}
	hexBlockHeader := hex.EncodeToString(headerBuf.Bytes())
	if coinbaseFlags == mining.CoinbaseFlagsStatic {
		reply <- &gbtResponse{hexBlockHeader, nil}
		return
	}
	mtxHex, err := marshal.MessageToHex(w.miner.template.Block.Transactions[0])
	if err != nil {
		reply <- &gbtResponse{nil, err}
		return
	}
	txHashs := []string{}
	for _, tx := range w.miner.template.TxMerklePath {
		txHashs = append(txHashs, tx.String())
	}
	var txWitnessRoot string
	if !w.miner.template.TxWitnessRoot.IsEqual(&hash.ZeroHash) {
		txWitnessRoot = w.miner.template.TxWitnessRoot.String()
	}

	reply <- &gbtResponse{&json.RemoteGBTResult{
		HeaderHex:     hexBlockHeader,
		CoinbaseTxHex: mtxHex,
		TxMerklePath:  txHashs,
		TxWitnessRoot: txWitnessRoot,
	}, nil}
}

func (w *RemoteWorker) GetRemoteGBTResult() *json.RemoteGBTResult {
	if atomic.LoadInt32(&w.shutdown) != 0 {
		return nil
	}

	var headerBuf bytes.Buffer
	err := w.miner.template.Block.Header.Serialize(&headerBuf)
	if err != nil {
		return nil
	}
	hexBlockHeader := hex.EncodeToString(headerBuf.Bytes())
	if w.miner.coinbaseFlags == mining.CoinbaseFlagsStatic {
		return &json.RemoteGBTResult{HeaderHex: hexBlockHeader}
	}
	mtxHex, err := marshal.MessageToHex(w.miner.template.Block.Transactions[0])
	if err != nil {
		return nil
	}
	txHashs := []string{}
	for _, tx := range w.miner.template.TxMerklePath {
		txHashs = append(txHashs, tx.String())
	}
	var txWitnessRoot string
	if !w.miner.template.TxWitnessRoot.IsEqual(&hash.ZeroHash) {
		txWitnessRoot = w.miner.template.TxWitnessRoot.String()
	}
	return &json.RemoteGBTResult{
		HeaderHex:     hexBlockHeader,
		CoinbaseTxHex: mtxHex,
		TxMerklePath:  txHashs,
		TxWitnessRoot: txWitnessRoot,
	}
}

func NewRemoteWorker(miner *Miner) *RemoteWorker {
	w := RemoteWorker{
		miner: miner,
	}
	return &w
}
