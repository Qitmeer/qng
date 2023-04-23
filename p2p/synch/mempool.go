/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
)

func (s *Sync) SendMempoolRequest(stream network.Stream, pe *peers.Peer) *common.Error {
	e := ReadRspCode(stream, s.p2p)
	if !e.Code.IsSuccess() {
		e.Add("mempool request rsp")
		return e
	}
	return nil
}

func (s *Sync) HandlerMemPool(ctx context.Context, msg interface{}, stream libp2pcore.Stream, pe *peers.Peer) *common.Error {
	mpr, ok := msg.(*pb.MemPoolRequest)
	if !ok {
		err := fmt.Errorf("message is not type *MsgFilterLoad")
		return ErrMessage(err)
	}

	curCount := uint64(s.p2p.TxMemPool().Count())
	if mpr.TxsNum != curCount && curCount != 0 {
		err := s.peerSync.OnMemPool(pe, &MsgMemPool{})
		if err != nil {
			return ErrMessage(err)
		}
	}
	return s.EncodeResponseMsg(stream, nil)
}

// OnMemPool is invoked when a peer receives a mempool qitmeer message.
// It creates and sends an inventory message with the contents of the memory
// pool up to the maximum inventory allowed per message.  When the peer has a
// bloom filter loaded, the contents are filtered accordingly.
func (ps *PeerSync) OnMemPool(sp *peers.Peer, msg *MsgMemPool) error {
	// Only allow mempool requests if the server has bloom filtering
	// enabled.
	services := sp.Services()
	if services&protocol.Bloom != protocol.Bloom {
		return fmt.Errorf("%s sent a filterclear request with no "+
			"filter loaded -- disconnecting", sp.GetID().String())
	}

	// Generate inventory message with the available transactions in the
	// transaction memory pool.  Limit it to the max allowed inventory
	// per message.  The NewMsgInvSizeHint function automatically limits
	// the passed hint to the maximum allowed, so it's safe to pass it
	// without double checking it here.
	txDescs := ps.sy.p2p.TxMemPool().TxDescs()

	invs := []*pb.InvVect{}
	for _, txDesc := range txDescs {
		// Either add all transactions when there is no bloom filter,
		// or only the transactions that match the filter when there is
		// one.
		filter := sp.Filter()
		if !filter.IsLoaded() || filter.MatchTxAndUpdate(txDesc.Tx) {
			invs = append(invs, NewInvVect(InvTypeTx, txDesc.Tx.Hash()))
		}
	}
	// Send the inventory message if there is anything to send.
	ps.sy.tryToSendInventoryRequest(sp, invs)

	return nil
}
