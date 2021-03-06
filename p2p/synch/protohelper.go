/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package synch

import (
	"bytes"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/meerdag"
	pb "github.com/Qitmeer/qng/p2p/proto/v1"
	"reflect"
)

func changePBHashsToHashs(hs []*pb.Hash) []*hash.Hash {
	result := []*hash.Hash{}
	for _, ha := range hs {
		h, err := hash.NewHash(ha.Hash)
		if err != nil {
			log.Warn(fmt.Sprintf("Can't NewHash:%v", ha.Hash))
			continue
		}
		result = append(result, h)
	}
	return result
}

func changePBHashToHash(ha *pb.Hash) *hash.Hash {
	h, err := hash.NewHash(ha.Hash)
	if err != nil {
		log.Warn(fmt.Sprintf("Can't NewHash:%v", ha.Hash))
		return nil
	}
	return h
}

func changeHashsToPBHashs(hs []*hash.Hash) []*pb.Hash {
	result := []*pb.Hash{}
	for _, ha := range hs {
		result = append(result, &pb.Hash{Hash: ha.Bytes()})
	}
	return result
}

func changePBTxToTx(tx *pb.Transaction) *types.Transaction {
	var transaction types.Transaction
	err := transaction.Deserialize(bytes.NewReader(tx.TxBytes))
	if err != nil {
		return nil
	}
	return &transaction
}

func getMessageString(message interface{}) string {
	str := fmt.Sprintf("%v:", reflect.TypeOf(message))
	switch msg := message.(type) {
	case *pb.ChainState:
		gh := changePBHashToHash(msg.GenesisHash)
		gs := changePBGraphStateToGraphState(msg.GraphState)
		str += fmt.Sprintf(" genesis:%s version:%d timestamp:%d services:%d disableRelayTx:%v useragent:%s",
			gh.String(), msg.ProtocolVersion, msg.Timestamp, msg.Services, msg.DisableRelayTx, string(msg.UserAgent))
		if gs != nil {
			str += fmt.Sprintf(" graphstate:%s", gs.String())
		}
		return str
	case pb.GetBlockDatas:
		locs := changePBHashsToHashs(msg.Locator)
		str += fmt.Sprintf(" locator:")
		for _, loc := range locs {
			str += " "
			str += loc.String()
		}
		return str
	case pb.GetBlocks:
		locs := changePBHashsToHashs(msg.Locator)
		str += fmt.Sprintf(" locator:")
		for _, loc := range locs {
			str += " "
			str += loc.String()
		}
		return str
	case *pb.GraphState:
		gs := changePBGraphStateToGraphState(msg)
		if gs != nil {
			str += fmt.Sprintf(" graphstate:%s", gs.String())
		}
		return str
	case *pb.Inventory:
		str += " invs:"
		for _, inv := range msg.Invs {
			str += fmt.Sprintf(" %d:%s", inv.Type, changePBHashToHash(inv.Hash))
		}
		return str
	case *pb.MemPoolRequest:
		str += fmt.Sprintf(" txsNum:%d", msg.TxsNum)
		return str
	case *pb.SyncDAG:
		mls := changePBHashsToHashs(msg.MainLocator)
		gs := changePBGraphStateToGraphState(msg.GraphState)
		str += fmt.Sprintf(" mainlocator:")
		for _, loc := range mls {
			str += " "
			str += loc.String()
		}
		if gs != nil {
			str += fmt.Sprintf(" graphstate:%s", gs.String())
		}
		return str
	case *pb.GetTxs:
		txs := changePBHashsToHashs(msg.Txs)
		str += fmt.Sprintf(" txs:")
		for _, tx := range txs {
			str += " "
			str += tx.String()
		}
		return str
	}
	str += fmt.Sprintf("%v", message)
	return str
}

func changePBGraphStateToGraphState(csgs *pb.GraphState) *meerdag.GraphState {
	if csgs == nil {
		return nil
	}
	gs := meerdag.NewGraphState()
	gs.SetTotal(uint(csgs.Total))
	gs.SetLayer(uint(csgs.Layer))
	gs.SetMainHeight(uint(csgs.MainHeight))
	gs.SetMainOrder(uint(csgs.MainOrder))
	tips := []*hash.Hash{}
	for _, tip := range csgs.Tips {
		h, err := hash.NewHash(tip.Hash)
		if err != nil {
			return nil
		}
		tips = append(tips, h)
	}
	gs.SetTips(tips)
	return gs
}
