package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/v2/core/serialization"
	"io"
)

const (
	AddressUTXOsSuffix = "-utxos"

	NormalUTXOType   = 0
	CoinbaseUTXOType = 1
	CLTVUTXOType     = 2
	TokenUTXOType    = 3
)

type AcctUTXO struct {
	typ     byte
	amount  uint64
	balance uint64
	final   bool
}

func (au *AcctUTXO) Encode(w io.Writer) error {
	err := s.WriteElements(w, au.typ)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, au.amount)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, au.balance)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, au.final)
	if err != nil {
		return err
	}
	return nil
}

func (au *AcctUTXO) Decode(r io.Reader) error {
	err := s.ReadElements(r, &au.typ)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &au.amount)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &au.balance)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &au.final)
	if err != nil {
		return err
	}
	return nil
}

func (au *AcctUTXO) String() string {
	return fmt.Sprintf("type=%s amount=%d balance=%d final=%v", au.TypeStr(), au.amount, au.balance, au.final)
}

func (au *AcctUTXO) TypeStr() string {
	switch au.typ {
	case NormalUTXOType:
		return "normal"
	case CoinbaseUTXOType:
		return "coinbase"
	case CLTVUTXOType:
		return "CLTV"
	case TokenUTXOType:
		return "token"
	}
	return "unknown"
}

func (au *AcctUTXO) SetCoinbase() {
	au.typ = CoinbaseUTXOType
	au.final = false
	au.balance = 0
}

func (au *AcctUTXO) IsCoinbase() bool {
	return au.typ == CoinbaseUTXOType
}

func (au *AcctUTXO) SetCLTV() {
	au.typ = CLTVUTXOType
	au.final = false
	au.balance = 0
}

func (au *AcctUTXO) IsCLTV() bool {
	return au.typ == CLTVUTXOType
}

func (au *AcctUTXO) FinalizeBalance(balance uint64) {
	au.balance = balance
	au.final = true
}

func (au *AcctUTXO) FinalizeBalanceByAmount() {
	au.balance = au.amount
	au.final = true
}

func (au *AcctUTXO) Lock() {
	if !au.IsCoinbase() && !au.IsCLTV() {
		return
	}
	au.balance = 0
	au.final = false
}

func (au *AcctUTXO) IsFinal() bool {
	return au.final
}

func NewAcctUTXO(amount uint64) *AcctUTXO {
	au := AcctUTXO{
		typ:     NormalUTXOType,
		amount:  amount,
		balance: 0,
		final:   true,
	}

	return &au
}
