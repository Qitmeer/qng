package notify

import "github.com/libp2p/go-libp2p/core/peer"

type NotifyData struct {
	Data    interface{}
	Filters []peer.ID
}

func (nd *NotifyData) IsFilter(pid peer.ID) bool {
	if len(nd.Filters) <= 0 {
		return false
	}

	for _, f := range nd.Filters {
		if f == pid {
			return true
		}
	}
	return false
}
