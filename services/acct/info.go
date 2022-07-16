package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

const (
	CurrentAcctInfoVersion = 0
)

type AcctInfo struct {
	version     uint32
	updateDAGID uint32
	addrTotal   uint32
}

func (ai *AcctInfo) Encode(w io.Writer) error {
	err := s.WriteElements(w, ai.version)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ai.updateDAGID)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ai.addrTotal)
	if err != nil {
		return err
	}
	return nil
}

func (ai *AcctInfo) Decode(r io.Reader) error {
	err := s.ReadElements(r, &ai.version)
	if err != nil {
		return err
	}

	err = s.ReadElements(r, &ai.updateDAGID)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &ai.addrTotal)
	if err != nil {
		return err
	}
	return nil
}

func (ai *AcctInfo) String() string {
	return fmt.Sprintf("version=%d dagid=%d addrtotal=%d", ai.version, ai.updateDAGID, ai.addrTotal)
}

func NewAcctInfo() *AcctInfo {
	ai := AcctInfo{
		version: CurrentAcctInfoVersion,
	}

	return &ai
}
