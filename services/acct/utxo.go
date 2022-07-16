package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

const (
	AddressUTXOsSuffix = "-utxos"

	AvailableUTXOState = 0
	LockedUTXOState    = 1
)

type AcctUTXO struct {
	state   byte
	balance uint64
}

func (au *AcctUTXO) Encode(w io.Writer) error {
	err := s.WriteElements(w, au.state)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, au.balance)
	if err != nil {
		return err
	}
	return nil
}

func (au *AcctUTXO) Decode(r io.Reader) error {
	err := s.ReadElements(r, &au.state)
	if err != nil {
		return err
	}

	err = s.ReadElements(r, &au.balance)
	if err != nil {
		return err
	}
	return nil
}

func (au *AcctUTXO) String() string {
	return fmt.Sprintf("state=%d balance=%d", au.state, au.balance)
}

func NewAcctUTXO() *AcctUTXO {
	au := AcctUTXO{
		state: AvailableUTXOState,
	}

	return &au
}
