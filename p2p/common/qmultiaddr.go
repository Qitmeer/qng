package common

import "github.com/multiformats/go-multiaddr"

type QMultiaddr struct {
	ma multiaddr.Multiaddr
}

func (qm *QMultiaddr) String() string {
	return qm.ma.String()
}

func QMultiAddrFromString(address string) (*QMultiaddr, error) {
	ma,err:=multiaddr.NewMultiaddr(address)
	if err != nil {
		return nil,err
	}
	return &QMultiaddr{ma: ma},nil
}
