package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

type AcctBalance struct {
	normal  uint64
	norUTXONum uint32
	locked     uint64
	locUTXONum uint32
}

func (ab *AcctBalance) Encode(w io.Writer) error {
	err := s.WriteElements(w, ab.normal)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ab.norUTXONum)
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
	err := s.ReadElements(r, &ab.normal)
	if err != nil {
		return err
	}

	err = s.ReadElements(r, &ab.norUTXONum)
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
	return fmt.Sprintf("normal=%d norUTXONum=%d locked=%d locUTXONum=%d",
		ab.normal, ab.norUTXONum, ab.locked, ab.locUTXONum)
}

func (ab *AcctBalance) IsEmpty() bool {
	return ab.norUTXONum ==0 && ab.locUTXONum == 0
}

func NewAcctBalance(normal uint64, norUTXONum uint32, locked uint64, locUTXONum uint32) *AcctBalance {
	ab := AcctBalance{
		normal:  normal,
		norUTXONum: norUTXONum,
		locked:     locked,
		locUTXONum: locUTXONum,
	}
	return &ab
}
