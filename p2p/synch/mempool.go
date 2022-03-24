/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/core/protocol"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	libp2pcore "github.com/libp2p/go-libp2p-core"
)

func (s *Sync) SendMempoolRequest(ctx context.Context, pe *peers.Peer) error {
	ctx, cancel := context.WithTimeout(ctx, ReqTimeout)
	defer cancel()

	stream, err := s.Send(ctx, &pb.MemPoolRequest{TxsNum:uint64(s.p2p.TxMemPool().Count())}, RPCMemPool, pe.GetID())
	if err != nil {
		return err
	}
	defer func() {
		if err := stream.Reset(); err != nil {
			log.Error(fmt.Sprintf("Failed to reset stream with protocol %s,%v", stream.Protocol(), err))
		}
	}()

	code, errMsg, err := ReadRspCode(stream, s.Encoding())
	if err != nil {
		return err
	}

	if !code.IsSuccess() {
		return errors.New(errMsg)
	}
	return nil
}

func (s *Sync) HandlerMemPool(ctx context.Context, msg interface{}, stream libp2pcore.Stream) *common.Error {
	ctx, cancel := context.WithTimeout(ctx, HandleTimeout)
	var err error
	defer func() {
		cancel()
	}()
	pe := s.peers.Get(stream.Conn().RemotePeer())
	if pe == nil {
		return ErrPeerUnknown
	}
	mpr, ok := msg.(*pb.MemPoolRequest)
	if !ok {
		err = fmt.Errorf("message is not type *MsgFilterLoad")
		return ErrMessage(err)
	}
	curCount:=uint64(s.p2p.TxMemPool().Count())
	if mpr.TxsNum == curCount || curCount == 0 {
		return nil
	}
	s.peerSync.msgChan <- &OnMsgMemPool{pe: pe, data: &MsgMemPool{}}

	return nil
}

// OnMemPool is invoked when a peer receives a mempool qitmeer message.
// It creates and sends an inventory message with the contents of the memory
// pool up to the maximum inventory allowed per message.  When the peer has a
// bloom filter loaded, the contents are filtered accordingly.
func (ps *PeerSync) OnMemPool(sp *peers.Peer, msg *MsgMemPool) {
	// Only allow mempool requests if the server has bloom filtering
	// enabled.
	services := sp.Services()
	if services&protocol.Bloom != protocol.Bloom {
		log.Debug(fmt.Sprintf("%s sent a filterclear request with no "+
			"filter loaded -- disconnecting", sp.GetID().String()))
		ps.Disconnect(sp)
		return
	}

	// Generate inventory message with the available transactions in the
	// transaction memory pool.  Limit it to the max allowed inventory
	// per message.  The NewMsgInvSizeHint function automatically limits
	// the passed hint to the maximum allowed, so it's safe to pass it
	// without double checking it here.
	txDescs := ps.sy.p2p.TxMemPool().TxDescs()

	invs:=[]*pb.InvVect{}
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
	ps.sy.tryToSendInventoryRequest(sp,invs)
}
