/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"context"
	"fmt"
	"github.com/Qitmeer/qng/p2p/common"
	"github.com/Qitmeer/qng/p2p/encoder"
	"github.com/Qitmeer/qng/p2p/peers"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"github.com/Qitmeer/qng/params"
	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"io"
	"reflect"
	"strings"
	"time"
)

const (

	// RPCGoodByeTopic defines the topic for the goodbye rpc method.
	RPCGoodByeTopic = "/qitmeer/req/goodbye/1"
	// RPCPingTopic defines the topic for the ping rpc method.
	RPCPingTopic = "/qitmeer/req/ping/1"
	// RPCMetaDataTopic defines the topic for the metadata rpc method.
	RPCMetaDataTopic = "/qitmeer/req/metadata/1"
	// RPCChainState defines the topic for the chain state rpc method.
	RPCChainState = "/qitmeer/req/chainstate/1"
	// RPCGetBlocks defines the topic for the get blocks rpc method.
	RPCGetBlocks = "/qitmeer/req/getblocks/1"
	// RPCGetBlockDatas defines the topic for the get blocks rpc method.
	RPCGetBlockDatas = "/qitmeer/req/getblockdatas/1"
	// RPCGetBlocks defines the topic for the get blocks rpc method.
	RPCSyncDAG = "/qitmeer/req/syncdag/1"
	// RPCTransaction defines the topic for the transaction rpc method.
	RPCTransaction = "/qitmeer/req/transaction/1"
	// RPCInventory defines the topic for the inventory rpc method.
	RPCInventory = "/qitmeer/req/inventory/1"
	// RPCGraphState defines the topic for the graphstate rpc method.
	RPCGraphState = "/qitmeer/req/graphstate/1"
	// RPCSyncQNR defines the topic for the syncqnr rpc method.
	RPCSyncQNR = "/qitmeer/req/syncqnr/1"
	// RPCGetMerkleBlocks defines the topic for the get merkle blocks rpc method.
	RPCGetMerkleBlocks = "/qitmeer/req/getmerkles/1"
	// RPCFilterAdd defines the topic for the filter add rpc method.
	RPCFilterAdd = "/qitmeer/req/filteradd/1"
	// RPCFilterClear defines the topic for the filter add rpc method.
	RPCFilterClear = "/qitmeer/req/filterclear/1"
	// RPCFilterLoad defines the topic for the filter add rpc method.
	RPCFilterLoad = "/qitmeer/req/filterload/1"
	// RPCMemPool defines the topic for the mempool rpc method.
	RPCMemPool = "/qitmeer/req/mempool/1"
	// RPCMemPool defines the topic for the getdata rpc method.
	RPCGetData = "/qitmeer/req/getdata/1"
)

// Time to first byte timeout. The maximum time to wait for first byte of
// request response (time-to-first-byte). The client is expected to give up if
// they don't receive the first byte within 20 seconds.
const TtfbTimeout = 20 * time.Second

// rpcHandler is responsible for handling and responding to any incoming message.
// This method may return an error to internal monitoring, but the error will
// not be relayed to the peer.
type rpcHandler func(context.Context, interface{}, libp2pcore.Stream, *peers.Peer) *common.Error

// RespTimeout is the maximum time for complete response transfer.
const RespTimeout = 20 * time.Second

// ReqTimeout is the maximum time for complete request transfer.
const ReqTimeout = 20 * time.Second

// HandleTimeout is the maximum time for complete handler.
const HandleTimeout = 20 * time.Second

type Sync struct {
	peers        *peers.Status
	peerSync     *PeerSync
	p2p          peers.P2P
	PeerInterval time.Duration
	LANPeers     map[peer.ID]struct{}

	disconnectionNotify *network.NotifyBundle
	connectionNotify    *network.NotifyBundle
}

func (s *Sync) Start() error {
	s.registerHandlers()

	s.AddConnectionHandler()
	s.AddDisconnectionHandler()

	s.maintainPeerStatuses()

	return s.peerSync.Start()
}

func (s *Sync) Stop() error {
	if s.connectionNotify != nil {
		s.p2p.Host().Network().StopNotify(s.connectionNotify)
	}
	if s.disconnectionNotify != nil {
		s.p2p.Host().Network().StopNotify(s.disconnectionNotify)
	}
	return s.peerSync.Stop()
}

func (s *Sync) registerHandlers() {
	s.registerRPCHandlers()
	//s.registerSubscribers()
}

// registerRPCHandlers for p2p RPC.
func (s *Sync) registerRPCHandlers() {

	s.registerRPC(
		RPCGoodByeTopic,
		new(uint64),
		s.goodbyeRPCHandler,
	)

	s.registerRPC(
		RPCPingTopic,
		new(uint64),
		s.pingHandler,
	)

	s.registerRPC(
		RPCMetaDataTopic,
		nil,
		s.metaDataHandler,
	)

	s.registerRPC(
		RPCChainState,
		&pb.ChainState{},
		s.chainStateHandler,
	)

	s.registerRPC(
		RPCGetBlocks,
		&pb.GetBlocks{},
		s.getBlocksHandler,
	)

	s.registerRPC(
		RPCGetBlockDatas,
		&pb.GetBlockDatas{},
		s.getBlockDataHandler,
	)

	s.registerRPC(
		RPCSyncDAG,
		&pb.SyncDAG{},
		s.syncDAGHandler,
	)

	s.registerRPC(
		RPCTransaction,
		&pb.GetTxs{},
		s.txHandler,
	)

	s.registerRPC(
		RPCInventory,
		&pb.Inventory{},
		s.inventoryHandler,
	)

	s.registerRPC(
		RPCGraphState,
		&pb.GraphState{},
		s.graphStateHandler,
	)

	s.registerRPC(
		RPCSyncQNR,
		&pb.SyncQNR{},
		s.QNRHandler,
	)

	s.registerRPC(
		RPCGetMerkleBlocks,
		&pb.MerkleBlockRequest{},
		s.getMerkleBlockDataHandler,
	)

	s.registerRPC(
		RPCFilterAdd,
		&pb.FilterAddRequest{},
		s.HandlerFilterMsgAdd,
	)

	s.registerRPC(
		RPCFilterClear,
		&pb.FilterClearRequest{},
		s.HandlerFilterMsgClear,
	)

	s.registerRPC(
		RPCFilterLoad,
		&pb.FilterLoadRequest{},
		s.HandlerFilterMsgLoad,
	)

	s.registerRPC(
		RPCMemPool,
		&pb.MemPoolRequest{},
		s.HandlerMemPool,
	)

	s.registerRPC(
		RPCGetData,
		&pb.Inventory{},
		s.GetDataHandler,
	)
}

// registerRPC for a given topic with an expected protobuf message type.
func (s *Sync) registerRPC(topic string, base interface{}, handle rpcHandler) {
	RegisterRPC(s.p2p, topic, base, handle)
}

func (s *Sync) Send(pe *peers.Peer, protocol string, message interface{}) (interface{}, error) {
	if !s.peerSync.IsRunning() {
		return nil, fmt.Errorf("No run PeerSync\n")
	}

	log.Trace("Send message", "protocol", getProtocol(s.p2p, protocol), "peer", pe.IDWithAddress())

	ctx, cancel := context.WithTimeout(s.p2p.Context(), ReqTimeout)
	defer cancel()

	var e *common.Error
	stream, e := Send(ctx, s.p2p, message, protocol, pe)
	if e != nil && !e.Code.IsSuccess() {
		processReqError(e, stream, pe)
		return nil, e.ToError()
	}

	var ret interface{}
	switch protocol {
	case RPCChainState:
		e = s.sendChainStateRequest(stream, pe)
	case RPCGoodByeTopic:
		e = s.sendGoodByeMessage(message, pe)
	case RPCGetBlockDatas:
		ret, e = s.sendGetBlockDataRequest(stream, pe)
	case RPCGetBlocks:
		ret, e = s.sendGetBlocksRequest(stream, pe)
	case RPCGraphState:
		ret, e = s.sendGraphStateRequest(stream, pe)
	case RPCInventory:
		e = s.sendInventoryRequest(stream, pe)
	case RPCMemPool:
		e = s.SendMempoolRequest(stream, pe)
	case RPCMetaDataTopic:
		ret, e = s.sendMetaDataRequest(stream, pe)
	case RPCPingTopic:
		e = s.SendPingRequest(stream, pe)
	case RPCSyncDAG:
		ret, e = s.sendSyncDAGRequest(stream, pe)
	case RPCSyncQNR:
		ret, e = s.sendQNRRequest(stream, pe)
	case RPCTransaction:
		ret, e = s.sendTxRequest(stream, pe)
	default:
		return nil, fmt.Errorf("Can't support:%s", protocol)
	}
	processReqError(e, stream, pe)

	if e != nil && !e.Code.IsSuccess() {
		return nil, e.ToError()
	}
	return ret, nil
}

func (s *Sync) PeerSync() *PeerSync {
	return s.peerSync
}

// Peers returns the peer status interface.
func (s *Sync) Peers() *peers.Status {
	return s.peers
}

func (s *Sync) Encoding() encoder.NetworkEncoding {
	return s.p2p.Encoding()
}

// SetStreamHandler sets the protocol handler on the p2p host multiplexer.
// This method is a pass through to libp2pcore.Host.SetStreamHandler.
func (s *Sync) SetStreamHandler(topic string, handler network.StreamHandler) {
	s.p2p.Host().SetStreamHandler(protocol.ID(topic), handler)
}

func (s *Sync) EncodeResponseMsg(stream libp2pcore.Stream, msg interface{}) *common.Error {
	return EncodeResponseMsg(s.p2p, stream, msg, common.ErrNone)
}

func (s *Sync) EncodeResponseMsgPro(stream libp2pcore.Stream, msg interface{}, retCode common.ErrorCode) *common.Error {
	return EncodeResponseMsg(s.p2p, stream, msg, retCode)
}

func NewSync(p2p peers.P2P) *Sync {
	sy := &Sync{p2p: p2p, peers: peers.NewStatus(p2p),
		PeerInterval: params.ActiveNetParams.TargetTimePerBlock * 2,
		LANPeers:     map[peer.ID]struct{}{}}
	sy.peerSync = NewPeerSync(sy)

	for _, pid := range p2p.Config().LANPeers {
		peid, err := peer.Decode(pid)
		if err != nil {
			log.Warn(fmt.Sprintf("LANPeers configuration error:%s", pid))
			continue
		}
		sy.LANPeers[peid] = struct{}{}
	}
	return sy
}

// registerRPC for a given topic with an expected protobuf message type.
func RegisterRPC(rpc peers.P2PRPC, basetopic string, base interface{}, handle rpcHandler) {
	topic := getProtocol(rpc, basetopic)

	rpc.Host().SetStreamHandler(protocol.ID(topic), func(stream network.Stream) {
		if !rpc.IsRunning() {
			log.Error("PeerSync is not running")
			return
		}
		ctx, cancel := context.WithTimeout(rpc.Context(), RespTimeout)
		defer cancel()

		SetRPCStreamDeadlines(stream)

		pe := rpc.Peers().Fetch(stream.Conn().RemotePeer())
		pe.UpdateAddrDir(nil, stream.Conn().RemoteMultiaddr(), stream.Conn().Stat().Direction)

		log.Trace("Stream handler", "protocol", topic, "peer", pe.IDWithAddress())

		// Given we have an input argument that can be pointer or [][32]byte, this gives us
		// a way to check for its reflect.Kind and based on the result, we can decode
		// accordingly.
		var msg interface{}
		if base != nil {
			t := reflect.TypeOf(base)
			var ty reflect.Type
			if t.Kind() == reflect.Ptr {
				ty = t.Elem()
			} else {
				ty = t
			}
			msgT := reflect.New(ty)
			msg = msgT.Interface()

			err := DecodeMessage(stream, rpc, msg)
			if err != nil {
				e := common.NewError(common.ErrStreamRead, err)
				// Debug logs for goodbye errors
				if strings.Contains(topic, RPCGoodByeTopic) {
					e.Add("Failed to decode goodbye stream message")
				} else {
					e.Add("Failed to decode stream message")
				}
				processRspError(ctx, e, stream, rpc, pe)
				return
			} else {
				size := rpc.Encoding().GetSize(msg)
				rpc.IncreaseBytesRecv(stream.Conn().RemotePeer(), size)
			}
		}
		processRspError(ctx, handle(ctx, msg, stream, pe), stream, rpc, pe)
	})
}

func processRspError(ctx context.Context, e *common.Error, stream network.Stream, rpc peers.P2PRPC, pe *peers.Peer) bool {
	streamOK := true
	if e == nil {
		e = common.NewSuccess()
	}
	if e.Code.IsStream() {
		streamOK = false
	} else if !e.Code.IsDAGConsensus() && !e.Code.IsSuccess() {
		resp, err := generateErrorResponse(e, rpc.Encoding())
		if err != nil {
			e.Add(fmt.Sprintf("Failed to generate a response error:%v", err))
			streamOK = false
		} else {
			_, err := stream.Write(resp)
			if err != nil {
				e.Add(fmt.Sprintf("Failed to write to stream:%v", err))
				streamOK = false
			}
		}
	}
	if streamOK {
		err := closeWriteStream(stream)
		if err != nil {
			e.AddError(err)
		}
		if !e.Code.IsSuccess() || err != nil {
			pe.IncrementBadResponses(e)
		}
		select {
		case <-time.After(RespTimeout):
		case <-ctx.Done():
		}
		err = closeStream(stream)
		if err != nil {
			e.AddError(err)
			pe.IncrementBadResponses(e)
		}
	} else {
		err := resetStream(stream)
		if err != nil {
			e.AddError(err)
		}
		pe.IncrementBadResponses(e)
	}
	return streamOK
}

func processReqError(e *common.Error, stream network.Stream, pe *peers.Peer) bool {
	streamOK := true
	if e == nil {
		e = common.NewSuccess()
	}
	if e.Code.IsStream() {
		streamOK = false
	}
	if streamOK {
		err := closeStream(stream)
		if err != nil {
			e.AddError(err)
		}
		if !e.Code.IsSuccess() || err != nil {
			pe.IncrementBadResponses(e)
		}
	} else {
		err := resetStream(stream)
		if err != nil {
			e.AddError(err)
		}
		pe.IncrementBadResponses(e)
	}
	return streamOK
}

// Send a message to a specific peer. The returned stream may be used for reading, but has been
// closed for writing.
func Send(ctx context.Context, rpc peers.P2PRPC, message interface{}, baseTopic string, pe *peers.Peer) (network.Stream, *common.Error) {
	curState := rpc.Host().Network().Connectedness(pe.GetID())
	if curState == network.CannotConnect {
		return nil, common.NewErrorStr(common.ErrLibp2pConnect, curState.String())
	}

	topic := getProtocol(rpc, baseTopic)

	var deadline = ReqTimeout + RespTimeout
	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	stream, err := rpc.Host().NewStream(ctx, pe.GetID(), protocol.ID(topic))
	if err != nil {
		return nil, common.NewErrorStr(common.ErrStreamBase, fmt.Sprintf("open stream on topic %v failed", topic))
	}
	SetRPCStreamDeadlines(stream)
	// do not encode anything if we are sending a metadata request
	if message == nil {
		// Close stream for writing.
		err = closeWriteStream(stream)
		if err != nil {
			return nil, common.NewError(common.ErrStreamBase, err)
		}
		return stream, nil
	}
	size, err := EncodeMessage(stream, rpc, message)
	if err != nil {
		return nil, common.NewErrorStr(common.ErrStreamWrite, fmt.Sprintf("encocde rpc message %v to stream failed:%v", getMessageString(message), err))
	}
	rpc.IncreaseBytesSent(pe.GetID(), size)
	// Close stream for writing.
	err = closeWriteStream(stream)
	if err != nil {
		return nil, common.NewError(common.ErrStreamBase, err)
	}
	return stream, nil
}

func EncodeResponseMsg(rpc peers.P2PRPC, stream libp2pcore.Stream, msg interface{}, retCode common.ErrorCode) *common.Error {
	_, err := stream.Write([]byte{byte(retCode)})
	if err != nil {
		return common.NewError(common.ErrStreamWrite, fmt.Errorf("%s, %s", err, retCode.String()))
	}
	if msg != nil {
		size, err := EncodeMessage(stream, rpc, msg)
		if err != nil {
			return common.NewError(common.ErrStreamWrite, fmt.Errorf("%s, %s", err, retCode.String()))
		}
		rpc.IncreaseBytesSent(stream.Conn().RemotePeer(), size)
	}
	return nil
}

func getTopic(baseTopic string) string {
	if baseTopic == RPCChainState || baseTopic == RPCGoodByeTopic {
		return baseTopic
	}
	return baseTopic + "/" + params.ActiveNetParams.Name
}

func getProtocol(rpc peers.P2PRPC, base string) string {
	return getTopic(base) + rpc.Encoding().ProtocolSuffix()
}

func processUnderlyingError(rpc peers.P2PRPC, pid peer.ID, err error) {
	if err.Error() == io.EOF.Error() {
		return
	}
	log.Info(fmt.Sprintf("An underlying error(%s), try to terminate the connection:%s", err, pid.String()))
	err = rpc.Disconnect(pid)
	if err != nil {
		log.Error(err.Error())
	}
}
