package synch

import (
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/blockchain"
	"github.com/Qitmeer/qng/core/blockchain/token"
	"github.com/Qitmeer/qng/core/blockchain/utxo"
	"github.com/Qitmeer/qng/core/event"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	ecommon "github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"time"
)

const (
	MinSnapSyncNumber     = 200
	SnapSyncReqInterval   = time.Second * 5
	MinSnapSyncChainDepth = meerdag.MaxSnapSyncTargetDepth
)

func (ps *PeerSync) loadSnapSync() {
	data, err := ps.sy.p2p.Consensus().DatabaseContext().GetSnapSync()
	if err != nil {
		log.Error(err.Error())
		return
	}
	if len(data) <= 0 {
		return
	}

	snapStatus := NewSnapStatusFromBytes(data)
	if snapStatus == nil {
		return
	}
	snapStatus.syncPoint = ps.Chain().BlockDAG().GetBlockById(uint(snapStatus.GetSyncPointID()))
	if snapStatus.syncPoint == nil {
		log.Error("Can't find snap status point", "id", snapStatus.GetSyncPointID())
		return
	}
	ps.snapStatus = snapStatus
	log.Info("Load snap sync info", "data", ps.snapStatus.ToString())
	err = ps.Chain().BeginSnapSyncing()
	if err != nil {
		log.Info("End snap-sync", "err", err.Error())
		return
	}
}

func (ps *PeerSync) saveSnapSync() {
	save := false
	if ps.IsSnapSync() && ps.snapStatus.IsInit() {
		save = true
	}

	if !save {
		err := ps.sy.p2p.Consensus().DatabaseContext().DeleteSnapSync()
		if err != nil {
			log.Error(err.Error())
			return
		}
		return
	}
	data, err := ps.snapStatus.Bytes()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = ps.sy.p2p.Consensus().DatabaseContext().PutSnapSync(data)
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("Save snap sync info", "data", ps.snapStatus.ToString())
}

func (ps *PeerSync) IsSnapSync() bool {
	return ps.snapStatus != nil
}

func (ps *PeerSync) GetSnapSyncInfo() *json.SnapSyncInfo {
	if !ps.IsSnapSync() {
		return nil
	}
	info := ps.snapStatus.ToInfo()
	info.Pivot = ps.Chain().MeerChain().GetPivot()
	return info
}

func (ps *PeerSync) startSnapSync() bool {
	if !ps.sy.p2p.IsSnap() {
		if ps.IsSnapSync() {
			log.Error("There is an unfinished snap-sync, please enable the snap service")
			return true
		}
		return false
	}

	best := ps.Chain().BestSnapshot()
	var bestPeer *peers.Peer
	if ps.IsSnapSync() {
		snapPeer, _ := ps.getSnapSyncPeer(0, nil, false)
		if snapPeer == nil {
			return true
		}
		bestPeer = snapPeer
	} else {
		bestPeer = ps.getBestPeer(NormalSyncMode, nil)
		if bestPeer == nil {
			return false
		}
		gs := bestPeer.GraphState()
		if gs.GetMainOrder() < best.GraphState.GetMainOrder()+MinSnapSyncChainDepth {
			return false
		}
		if !ps.isValidSyncPeer(bestPeer, SnapSyncMode) {
			snapPeer, change := ps.getSnapSyncPeer(ps.sy.p2p.Consensus().Config().NoSnapSyncPeerTimeout, nil, true)
			if snapPeer == nil {
				if change {
					return false
				}
				return true
			}
			bestPeer = snapPeer
		}
	}

	if !ps.IsRunning() {
		return true
	}
	// Start syncing from the best peer if one was selected.
	startTime, _ := ps.prepSync(bestPeer)
	defer func() {
		defer ps.processwg.Done()
		ps.SetSyncPeer(nil)
	}()

	if ps.IsSnapSync() {
		ps.snapStatus.ResetPeer(bestPeer.GetID())
		log.Info("Snap syncing continue", "cur", best.GraphState.String(), "status", ps.snapStatus.ToString(), "processID", ps.getProcessID())
	} else {
		ps.snapStatus = NewSnapStatus(bestPeer.GetID())
		log.Info("Snap syncing", "cur", best.GraphState.String(), "status", ps.snapStatus.ToString(), "processID", ps.getProcessID())
	}

	ps.sy.p2p.Consensus().Events().Send(event.New(event.DownloaderStart))
	// ------
	err := ps.Chain().BeginSnapSyncing()
	if err != nil {
		log.Info("End snap-sync", "err", err.Error())
		return true
	}
	add := 0

	endSnapSyncing := func() {
		ps.sy.p2p.Consensus().Events().Send(event.New(event.DownloaderEnd))
		if ps.snapStatus.IsCompleted() {
			log.Info("Snap-sync has ended", "spend", time.Since(startTime).Truncate(time.Second).String(), "add", add, "processID", ps.getProcessID())
		} else {
			log.Warn("Snap-sync illegal ended", "spend", time.Since(startTime).Truncate(time.Second).String(), "add", add, "processID", ps.getProcessID())
		}
		sp := ps.snapStatus.GetSyncPoint()
		if sp != nil {
			bestPeer.UpdateSyncPoint(sp.GetHash())
			log.Debug("Snap-sync update sync point", "point", sp.GetHash().String())
		}
		isCompleted := ps.snapStatus.IsCompleted()
		if isCompleted {
			ps.snapStatus = nil
		} else {
			go ps.TryAgainUpdateSyncPeer(false)
		}
		ps.saveSnapSync()
		if isCompleted {
			err := ps.sy.p2p.BlockChain().PrepareEnvironment()
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
	defer endSnapSyncing()

	ebc := ps.Chain().MeerChain().Ether().BlockChain()
	mc := ps.sy.p2p.BlockChain().MeerChain()
	if !ps.snapStatus.IsEVMCompleted() {
		curEVMTarget := ps.snapStatus.GetEVMTarget()
		if curEVMTarget != (ecommon.Hash{}) && ebc.GetBlockByHash(curEVMTarget) != nil {
			ps.snapStatus.CompleteEVM()
		}
		if mc.CurrentHeader().Number.Uint64()+MinSnapSyncNumber >= bestPeer.GetMeerState().Number {
			ps.snapStatus.CompleteEVM()
		}
	}
	var lastEvmTarget ecommon.Hash
	timeout := time.Minute * 5
	ticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

	for !ps.snapStatus.IsCompleted() {
		if !ps.IsRunning() {
			log.Warn("Snap-sync exit midway")
			return true
		}
		ret, pe := ps.trySyncSnapStatus(bestPeer)
		bestPeer = pe
		if ret == nil {
			log.Warn("Snap-sync can't get rsp data")
			return true
		}
		sds, err := ps.processRsp(ret)
		if err != nil {
			log.Warn(err.Error())
			continue
		}
		if len(sds) > 0 {
			latest, err := ps.Chain().ProcessBlockBySnap(sds)
			if err != nil {
				panic(err.Error())
			}
			ps.snapStatus.SetSyncPoint(latest)
			add += len(sds)
			log.Trace("Snap-sync", "point", latest.GetHash().String(), "data_num", len(sds), "total", add)
		}
		curEVMTarget := ps.snapStatus.GetEVMTarget()
		if (ps.snapStatus.IsPointCompleted() || lastEvmTarget == (ecommon.Hash{})) && !ps.snapStatus.IsEVMCompleted() && curEVMTarget != (ecommon.Hash{}) {
			if lastEvmTarget != curEVMTarget || lastEvmTarget == (ecommon.Hash{}) {
				if ebc.GetBlockByHash(curEVMTarget) != nil {
					ps.snapStatus.CompleteEVM()
				} else {
					err := mc.Downloader().SyncQngWaitPeers(curEVMTarget, ps.quit, timeout)
					if err != nil {
						log.Warn("Failed to trigger beacon sync", "err", err)
					}
				}
				lastEvmTarget = curEVMTarget
			}
			if ps.snapStatus.IsPointCompleted() && !ps.snapStatus.IsEVMCompleted() {
				select {
				case <-time.After(timeout):
					log.Debug("Try to compare target for snap-sync", "cur", curEVMTarget.String())
				case <-ticker.C:
					block := ebc.GetBlockByHash(curEVMTarget)
					if block != nil {
						ps.snapStatus.CompleteEVM()
						log.Info("Sync target reached", "number", block.NumberU64(), "hash", block.Hash())
					}
				case <-ps.quit:
					log.Warn("Snap-sync exit midway")
					return true
				}
			}
		}
	}

	ps.sy.p2p.BlockChain().EndSnapSyncing()
	return true
}

func (ps *PeerSync) trySyncSnapStatus(pe *peers.Peer) (*pb.SnapSyncRsp, *peers.Peer) {
	var ret *pb.SnapSyncRsp
	for ret == nil {
		if !ps.IsRunning() {
			return nil, pe
		}
		rsp := make(chan *pb.SnapSyncRsp)
		go ps.syncSnapStatus(pe, rsp)
		select {
		case ret = <-rsp:
			log.Debug("SnapSyncRsp", "peer", pe.GetID().String(), "result", ret != nil)
		case <-ps.quit:
			return nil, pe
		}
		if ret != nil {
			return ret, pe
		}
		log.Warn("Try change snap peer", "processID", ps.getProcessID())
		newPeer, _ := ps.getSnapSyncPeer(0, map[peer.ID]struct{}{pe.GetID(): struct{}{}}, false)
		if newPeer == nil {
			return nil, pe
		}
		pe = newPeer
		ps.snapStatus.ResetPeer(pe.GetID())
	}
	return ret, pe
}

func (ps *PeerSync) syncSnapStatus(pe *peers.Peer, rsp chan *pb.SnapSyncRsp) {
	req := &pb.SnapSyncReq{Locator: []*pb.Locator{}}

	targetBlock, stateRoot := ps.snapStatus.GetTarget()
	if targetBlock != nil {
		sp := ps.snapStatus.GetSyncPoint()
		req.Target = &pb.Locator{Block: &pb.Hash{Hash: targetBlock.Bytes()}, Root: &pb.Hash{Hash: stateRoot.Bytes()}}
		req.Locator = append(req.Locator, &pb.Locator{
			Block: &pb.Hash{Hash: sp.GetHash().Bytes()},
			Root:  &pb.Hash{Hash: sp.GetState().Root().Bytes()},
		})
		if targetBlock.IsEqual(sp.GetHash()) {
			log.Debug("Compare target for snap-sync")
		} else {
			log.Debug("Continue target for snap-sync")
		}
	} else {
		zeroH := &pb.Hash{Hash: hash.ZeroHash.Bytes()}
		req.Target = &pb.Locator{Block: zeroH, Root: zeroH}
		point := pe.SyncPoint()
		mainLocator, mainStateRoot := ps.dagSync.GetMainLocator(point, true)
		for i := 0; i < len(mainLocator); i++ {
			req.Locator = append(req.Locator, &pb.Locator{
				Block: &pb.Hash{Hash: mainLocator[i].Bytes()},
				Root:  &pb.Hash{Hash: mainStateRoot[i].Bytes()},
			})
		}
		log.Debug("Init target for snap-sync")
	}

	ret, err := pe.Request(SnapSyncReqMsg, req)
	if err != nil {
		log.Error(err.Error())
		select {
		case rsp <- nil:
		case <-ps.quit:
		}
		return
	}
	if ret != nil {
		select {
		case rsp <- ret.(*pb.SnapSyncRsp):
		case <-ps.quit:
		}
	}
}

func (ps *PeerSync) processRsp(ssr *pb.SnapSyncRsp) ([]*blockchain.SnapData, error) {
	if isLocatorEmpty(ssr.Target) {
		return nil, fmt.Errorf("No snap sync data")
	}
	targetBlock := changePBHashToHash(ssr.Target.Block)
	stateRoot := changePBHashToHash(ssr.Target.Root)
	evm := ecommon.BytesToHash(ssr.Evm.Hash)
	ps.snapStatus.SetTarget(targetBlock, stateRoot)
	ps.snapStatus.SetEVMTarget(evm)

	log.Trace("Snap-sync receive", "targetBlock", targetBlock.String(), "stateRoot", stateRoot.String(), "evm", evm.String(), "dataNum", len(ssr.Datas), "processID", ps.getProcessID())

	if len(ssr.Datas) <= 0 {
		return nil, nil
	}
	ret := []*blockchain.SnapData{}
	for _, data := range ssr.Datas {
		sd := blockchain.NewSnapData()
		if len(data.Block) > 0 {
			block, err := types.NewBlockFromBytes(data.Block)
			if err != nil {
				return nil, err
			}
			sd.SetBlock(block)
		}
		if len(data.Stxos) > 0 {
			sts, err := utxo.DeserializeSpendJournalEntry(data.Stxos)
			if err != nil {
				return nil, err
			}
			sd.SetStxos(sts)
		}
		if len(data.DagBlock) > 0 {
			dblock, err := ps.Chain().BlockDAG().NewBlockFromBytes(data.DagBlock)
			if err != nil {
				return nil, err
			}
			var dagBIDs map[uint]*hash.Hash
			if len(data.DagBIDs) > 0 {
				dagBIDs = map[uint]*hash.Hash{}
				for _, bid := range data.DagBIDs {
					dagBIDs[uint(bid.Id)] = changePBHashToHash(bid.Block)
				}
			}
			sd.SetDAGBlock(dblock, dagBIDs)
		}
		sd.SetMain(data.Main)
		if len(data.TokenState) > 0 {
			ts, err := token.NewTokenStateFromBytes(data.TokenState)
			if err != nil {
				return nil, err
			}
			sd.SetTokenState(ts)
		}
		if !isZeroPBHash(data.PrevTSHash) {
			sd.SetPrevTSHash(changePBHashToHash(data.PrevTSHash))
		}
		ret = append(ret, sd)
	}
	return ret, nil
}

func (ps *PeerSync) getSnapSyncPeer(timeout int, exclude map[peer.ID]struct{}, pre bool) (*peers.Peer, bool) {
	start := time.Now()
	var pe *peers.Peer
	for pe == nil {
		pe = ps.getBestPeer(SnapSyncMode, exclude)
		if pre {
			if pe != nil {
				best := ps.Chain().BestSnapshot()
				if pe.GraphState().GetMainOrder() < best.GraphState.GetMainOrder()+MinSnapSyncChainDepth {
					pe = nil
				}
			}
		}
		if pe == nil {
			if timeout > 0 {
				if time.Since(start) > time.Duration(timeout)*time.Second {
					return nil, true
				}
			}
			log.Debug("Try to get snap-sync peer", "cost", time.Since(start).String())
			time.Sleep(SnapSyncReqInterval)
		}
		if !ps.IsRunning() {
			return nil, false
		}
	}
	return pe, false
}

func (s *Sync) snapSyncHandler(id uint64, msg interface{}, pe *peers.Peer) error {
	m, ok := msg.(*pb.SnapSyncReq)
	if !ok {
		return fmt.Errorf("message is not type *SnapSyncReq")
	}

	blocks, target, err := s.peerSync.dagSync.CalcSnapSyncBlocks(changePBLocatorsToLocators(m.Locator), MaxBlockLocatorsPerMsg, changePBLocatorToLocator(m.Target), s.peerSync.Chain().MeerChain().CheckState)
	if err != nil {
		return err
	}
	log.Debug("Received Snap-sync request", "reqId", id, "peer", pe.GetID().String(), "target", target.GetHash().String(), "number", target.GetState().GetEVMNumber(), "pivot", meerdag.CalculatePivot(target.GetState().GetEVMNumber()))
	rsp := &pb.SnapSyncRsp{Datas: []*pb.TransferData{}}
	rsp.Target = &pb.Locator{
		Block: &pb.Hash{Hash: target.GetHash().Bytes()},
		Root:  &pb.Hash{Hash: target.GetState().Root().Bytes()},
	}
	rsp.Evm = &pb.Hash{Hash: target.GetState().GetEVMHash().Bytes()}

	if len(blocks) > 0 {
		for _, block := range blocks {
			data := &pb.TransferData{DagBIDs: []*pb.BlockID{}}
			blkBytes, err := s.p2p.BlockChain().FetchBlockBytesByHash(block.GetHash())
			if err != nil {
				return err
			}
			data.Block = blkBytes

			stxo, err := s.p2p.BlockChain().DB().GetSpendJournal(block.GetHash())
			if err != nil {
				return err
			}
			data.Stxos = stxo

			// process DAG block data
			data.DagBlock = block.Bytes()
			pblock, ok := block.(*meerdag.PhantomBlock)
			if !ok {
				err := fmt.Errorf("Not phantom block %s", block.GetHash().String())
				return err
			}
			if pblock.GetBlueDiffAnticoneSize() > 0 {
				for k, _ := range pblock.GetBlueDiffAnticone().GetMap() {
					bh := s.p2p.BlockChain().BlockDAG().GetBlockHash(k)
					if bh == nil {
						return fmt.Errorf("No block hash:%d", k)
					}
					data.DagBIDs = append(data.DagBIDs, &pb.BlockID{
						Block: &pb.Hash{Hash: bh.Bytes()},
						Id:    uint32(k),
					})
				}
			}
			if pblock.GetRedDiffAnticoneSize() > 0 {
				for k, _ := range pblock.GetRedDiffAnticone().GetMap() {
					bh := s.p2p.BlockChain().BlockDAG().GetBlockHash(k)
					if bh == nil {
						return fmt.Errorf("No block hash:%d", k)
					}
					data.DagBIDs = append(data.DagBIDs, &pb.BlockID{
						Block: &pb.Hash{Hash: bh.Bytes()},
						Id:    uint32(k),
					})
				}
			}

			data.Main = s.p2p.BlockChain().BlockDAG().IsOnMainChain(block.GetID())
			ts := s.p2p.BlockChain().GetTokenState(uint32(block.GetID()))
			if ts != nil {
				prevTSHash, err := meerdag.DBGetDAGBlockHashByID(s.p2p.BlockChain().DB(), uint64(ts.PrevStateID))
				if err != nil {
					return err
				}
				serializedData, err := ts.Serialize()
				if err != nil {
					return err
				}
				data.TokenState = serializedData
				data.PrevTSHash = &pb.Hash{
					Hash: prevTSHash.Bytes(),
				}
			} else {
				data.PrevTSHash = &pb.Hash{
					Hash: hash.ZeroHash.Bytes(),
				}
			}
			if uint64(rsp.SizeSSZ()+data.SizeSSZ()+BLOCKDATA_SSZ_HEAD_SIZE) >= s.p2p.Encoding().GetMaxChunkSize() {
				break
			}
			rsp.Datas = append(rsp.Datas, data)
		}
	}
	return pe.Respond(SnapSyncRspMsg, rsp, id)
}
