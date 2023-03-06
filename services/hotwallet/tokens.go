package hotwallet

import (
	json2 "encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/services/hotwallet/walletdb"
	"sync"
)

type QitmeerToken struct {
	tokens map[types.CoinID]*json.TokenState
	lock   sync.RWMutex
}

func NewQitmeerToken(ns walletdb.ReadWriteBucket) *QitmeerToken {
	tokens := make(map[types.CoinID]*json.TokenState, 0)
	_ = ns.ForEach(func(k, v []byte) error {
		token, err := DecodeToken(v)
		if err == nil {
			tokens[types.CoinID(token.CoinId)] = token
			types.CoinNameMap[types.CoinID(token.CoinId)] = token.CoinName
			return nil
		} else {
			return err
		}
	})

	return &QitmeerToken{
		tokens: tokens,
		lock:   sync.RWMutex{},
	}
}

func (q *QitmeerToken) Add(t json.TokenState) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.tokens[types.CoinID(t.CoinId)] = &t
}

func (q *QitmeerToken) GetToken(coin types.CoinID) (*json.TokenState, error) {
	q.lock.RLock()
	defer q.lock.RUnlock()

	token, ok := q.tokens[coin]
	if ok {
		return token, nil
	}
	return nil, fmt.Errorf("coin %d dose not exist", coin)
}

func (q *QitmeerToken) Encode() []byte {
	bytes, _ := json2.Marshal(q)
	return bytes
}

func EncodeToken(state json.TokenState) []byte {
	bytes, _ := json2.Marshal(state)
	return bytes
}

func DecodeToken(bytes []byte) (*json.TokenState, error) {
	var t = &json.TokenState{}
	err := json2.Unmarshal(bytes, t)
	return t, err
}
