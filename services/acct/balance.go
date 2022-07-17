package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

type AcctBalance struct {
	available  uint64
	avaUTXONum uint32
	locked     uint64
	locUTXONum uint32
}

func (ab *AcctBalance) Encode(w io.Writer) error {
	err := s.WriteElements(w, ab.available)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ab.avaUTXONum)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ab.locked)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ab.locUTXONum)
	if err != nil {
		return err
	}
	return nil
}

func (ab *AcctBalance) Decode(r io.Reader) error {
	err := s.ReadElements(r, &ab.available)
	if err != nil {
		return err
	}

	err = s.ReadElements(r, &ab.avaUTXONum)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &ab.locked)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &ab.locUTXONum)
	if err != nil {
		return err
	}
	return nil
}

func (ab *AcctBalance) String() string {
	return fmt.Sprintf("available=%d avaUTXONum=%d locked=%d locUTXONum=%d",
		ab.available, ab.avaUTXONum, ab.locked, ab.locUTXONum)
}

func NewAcctBalance(available uint64, avaUTXONum uint32, locked uint64, locUTXONum uint32) *AcctBalance {
	ab := AcctBalance{
		available:  available,
		avaUTXONum: avaUTXONum,
		locked:     locked,
		locUTXONum: locUTXONum,
	}
	return &ab
}
