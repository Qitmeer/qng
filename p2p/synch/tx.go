/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync/atomic"
)

const TXDATA_SSZ_HEAD_SIZE = 4

func (s *Sync) sendTxRequest(stream network.Stream, pe *peers.Peer) (*pb.Transactions, *common.Error) {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("tx request rsp")
		return nil, e
	}
	msg := &pb.Transactions{}
	if err := DecodeMessage(stream, s.p2p, msg); err != nil {
		return nil, common.NewError(common.ErrStreamRead, err)
	}
	return msg, nil
}

func (s *Sync) txHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {

	m, ok := msg.(*pb.GetTxs)
	if !ok {
		err := fmt.Errorf("message is not type *pb.Transaction")
		return ErrMessage(err)
	}

	txs, err := s.p2p.TxMemPool().FetchTransactions(changePBHashsToHashs(m.Txs))
	if err != nil {
		log.Trace(fmt.Sprintf("Unable to fetch txs %v from transaction pool : %v ", len(m.Txs), err))
		return ErrMessage(err)
	}

	pbtxs := &pb.Transactions{Txs: []*pb.Transaction{}}
	for _, tx := range txs {
		if len(pbtxs.Txs) >= MaxInvPerMsg {
			break
		}
		txbytes, err := tx.Tx.Serialize()
		if err != nil {
			log.Warn(err.Error())
			continue
		}
		pbtx := &pb.Transaction{TxBytes: txbytes}
		if uint64(pbtxs.SizeSSZ()+pbtx.SizeSSZ()+TXDATA_SSZ_HEAD_SIZE) >= s.p2p.Encoding().GetMaxChunkSize() {
			break
		}
		pbtxs.Txs = append(pbtxs.Txs, pbtx)
	}
	return s.EncodeResponseMsg(stream, pbtxs)
}

func (s *Sync) handleTxMsg(msg *pb.Transaction, pid peer.ID) (*hash.Hash, error) {
	tx := changePBTxToTx(msg)
	if tx == nil {
		return nil, fmt.Errorf("message is not type *pb.Transaction")
	}
	txh := tx.TxHash()
	// Process the transaction to include validation, insertion in the
	// memory pool, orphan handling, etc.
	allowOrphans := s.p2p.Config().MaxOrphanTxs > 0
	acceptedTxs, err := s.p2p.TxMemPool().ProcessTransaction(types.NewTx(tx), allowOrphans, true, true)
	if err != nil {
		return &txh, fmt.Errorf("Failed to process transaction %v: %v\n", tx.TxHash().String(), err.Error())
	}
	s.p2p.Notify().AnnounceNewTransactions(acceptedTxs, nil, []peer.ID{pid})

	return &txh, nil
}

func (ps *PeerSync) processGetTxs(pe *peers.Peer, otxs []*hash.Hash) error {
	if len(otxs) <= 0 {
		return nil
	}
	txsM := map[string]struct{}{}
	var gtxs *pb.GetTxs
	for i, txh := range otxs {
		if !ps.IsRunning() {
			return fmt.Errorf("No run PeerSync\n")
		}
		_, ok := txsM[txh.String()]
		if ok {
			continue
		}
		if ps.sy.p2p.TxMemPool().HaveTransaction(txh) {
			continue
		}
		//
		txsM[txh.String()] = struct{}{}

		if gtxs == nil {
			gtxs = &pb.GetTxs{Txs: []*pb.Hash{}}
		}
		gtxs.Txs = append(gtxs.Txs, &pb.Hash{Hash: txh.Bytes()})
		if len(gtxs.Txs) < MaxInvPerMsg && i < (len(otxs)-1) {
			continue
		}
		if len(gtxs.Txs) <= 0 {
			continue
		}
		ret, err := ps.sy.Send(pe, RPCTransaction, gtxs)
		if err != nil {
			return err
		}
		gtxs = nil
		//
		ptxs := ret.(*pb.Transactions)
		if len(ptxs.Txs) <= 0 {
			continue
		}
		for _, tx := range ptxs.Txs {
			if !ps.IsRunning() {
				return fmt.Errorf("No run PeerSync\n")
			}
			_, err := ps.sy.handleTxMsg(tx, pe.GetID())
			if err != nil {
				log.Debug(err.Error())
			}
		}
	}
	return nil
}

func (ps *PeerSync) getTxs(pe *peers.Peer, txs []*hash.Hash) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&ps.shutdown) != 0 {
		return
	}
	err := ps.processGetTxs(pe, txs)
	if err != nil {
		log.Debug(err.Error())
	}
}
