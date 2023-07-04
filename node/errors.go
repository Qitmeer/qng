package node

import "errors"

var (
	ErrNodeStopped = errors.New("node not started")
	ErrDatadirUsed = errors.New("datadir already used by another process")
)
