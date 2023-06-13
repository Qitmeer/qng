/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"strings"
)

// MaxBlockLocatorsPerMsg is the maximum number of block locator hashes allowed
// per message.
const MaxBlockLocatorsPerMsg = 2000

func (s *Sync) sendSyncDAGRequest(stream network.Stream, pe *peers.Peer) (*pb.SubDAG, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("sync DAG request rsp")
		return nil, e
	}

	msg := &pb.SubDAG{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}

func (s *Sync) syncDAGHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	m, ok := msg.(*pb.SyncDAG)
	if !ok {
		err := fmt.Errorf("message is not type *pb.Hash")
		return ErrMessage(err)
	}
	pe.UpdateGraphState(m.GraphState)

	gs := pe.GraphState()
	blocks, point := s.PeerSync().dagSync.CalcSyncBlocks(gs, changePBHashsToHashs(m.MainLocator), meerdag.SubDAGMode, MaxBlockLocatorsPerMsg)
	pe.UpdateSyncPoint(point)
	sd := &pb.SubDAG{SyncPoint: &pb.Hash{Hash: point.Bytes()}, GraphState: s.getGraphState(), Blocks: changeHashsToPBHashs(blocks)}

	return s.EncodeResponseMsg(stream, sd)
}

func debugSyncDAG(m *pb.SyncDAG) string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("SyncDAG: graphstate=(%v,%v,%v), ",
		m.GraphState.MainOrder, m.GraphState.MainHeight, m.GraphState.Layer,
	))
	sb.WriteString("locator=[")
	size := len(m.MainLocator)
	for i, h := range m.MainLocator {
		sb.WriteString(changePBHashToHash(h).String())
		if i+1 < size {
			sb.WriteString(",")
		}
	}
	sb.WriteString("]")
	sb.WriteString(fmt.Sprintf(", size=%d ", size))
	return sb.String()
}

func (ps *PeerSync) processSyncDAGBlocks(pe *peers.Peer) *ProcessResult {
	log.Trace(fmt.Sprintf("processSyncDAGBlocks peer=%v ", pe.GetID()), "processID", ps.processID)
	if !ps.isSyncPeer(pe) || !pe.IsConnected() {
		return nil
	}
	point := pe.SyncPoint()
	mainLocator := ps.dagSync.GetMainLocator(point)
	sd := &pb.SyncDAG{MainLocator: changeHashsToPBHashs(mainLocator), GraphState: ps.sy.getGraphState()}
	log.Trace(fmt.Sprintf("processSyncDAGBlocks sendSyncDAG point=%v, sd=%v", point.String(), debugSyncDAG(sd)), "processID", ps.processID)
	ret, err := ps.sy.Send(pe, RPCSyncDAG, sd)
	if err != nil {
		log.Trace(fmt.Sprintf("processSyncDAGBlocks err=%v ", err.Error()), "processID", ps.processID)
		return &ProcessResult{act: ProcessResultActionTryAgain}
	}
	subd := ret.(*pb.SubDAG)
	if ps.IsInterrupt() {
		return nil
	}
	log.Trace(fmt.Sprintf("processSyncDAGBlocks result graphstate=(%v,%v,%v), blocks=%v ",
		subd.GraphState.MainOrder, subd.GraphState.MainHeight, subd.GraphState.Layer,
		len(subd.Blocks)), "processID", ps.processID)
	pe.UpdateSyncPoint(changePBHashToHash(subd.SyncPoint))
	pe.UpdateGraphState(subd.GraphState)

	if len(subd.Blocks) <= 0 {
		pe.UpdateSyncPoint(ps.Chain().BlockDAG().GetGenesisHash())
		return &ProcessResult{act: ProcessResultActionTryAgain}
	}
	log.Trace(fmt.Sprintf("processSyncDAGBlocks do GetBlockDatas blocks=%v ", len(subd.Blocks)), "processID", ps.processID)
	return ps.processGetBlockDatas(pe, changePBHashsToHashs(subd.Blocks))
}
