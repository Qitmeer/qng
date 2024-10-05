package acct

import (
	"fmt"
	s "github.com/Qitmeer/qng/core/serialization"
	"io"
)

const (
	CurrentAcctInfoVersion = 3
)

type AcctInfo struct {
	version     uint32
	updateDAGID uint32
	total       uint32
	all         bool
	addrs       []string
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
	err = s.WriteElements(w, ai.total)
	if err != nil {
		return err
	}
	err = s.WriteElements(w, ai.all)
	if err != nil {
		return err
	}
	addrTotal := ai.GetAddrTotal()
	err = s.WriteElements(w, uint32(addrTotal))
	if err != nil {
		return err
	}
	if addrTotal > 0 {
		for _, addr := range ai.addrs {
			err = s.WriteElements(w, uint32(len(addr)))
			if err != nil {
				return err
			}
			_, err = w.Write([]byte(addr))
			if err != nil {
				return err
			}
		}
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
	err = s.ReadElements(r, &ai.total)
	if err != nil {
		return err
	}
	err = s.ReadElements(r, &ai.all)
	if err != nil {
		return err
	}
	addrTotal := uint32(0)
	err = s.ReadElements(r, &addrTotal)
	if err != nil {
		return err
	}
	if addrTotal > 0 {
		for i := 0; i < int(addrTotal); i++ {
			addrLen := uint32(0)
			err = s.ReadElements(r, &addrLen)
			if err != nil {
				return err
			}
			addrby := make([]byte, addrLen)
			_, err = r.Read(addrby)
			if err != nil {
				return err
			}
			ai.addrs = append(ai.addrs, string(addrby))
		}
	}
	return nil
}

func (ai *AcctInfo) String() string {
	return fmt.Sprintf("version=%d dagid=%d addrtotal=%d/%d allmode=%v", ai.version, ai.updateDAGID, ai.total, ai.GetAddrTotal(), ai.all)
}

func (ai *AcctInfo) IsCurrentVersion() bool {
	return ai.version == CurrentAcctInfoVersion
}

func (ai *AcctInfo) GetAddrTotal() int {
	return len(ai.addrs)
}

func (ai *AcctInfo) IsEmpty() bool {
	return ai.GetAddrTotal() <= 0
}

func (ai *AcctInfo) Has(addr string) bool {
	if ai.GetAddrTotal() <= 0 {
		return false
	}
	for _, ad := range ai.addrs {
		if ad == addr {
			return true
		}
	}
	return false
}

func (ai *AcctInfo) Add(addr string) {
	ai.addrs = append(ai.addrs, addr)
	log.Info("Add address for AccountManager", "addr", addr, "total", ai.GetAddrTotal())
}

func (ai *AcctInfo) Del(addr string) {
	if ai.GetAddrTotal() <= 0 {
		return
	}
	ret := []string{}
	for _, ad := range ai.addrs {
		if ad == addr {
			continue
		}
		ret = append(ret, ad)
	}
	ai.addrs = ret
	log.Info("Delete address for AccountManager", "addr", addr, "total", ai.GetAddrTotal())
}

func NewAcctInfo() *AcctInfo {
	ai := AcctInfo{
		version: CurrentAcctInfoVersion,
		addrs:   []string{},
		total:   0,
	}

	return &ai
}
