/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package event

const (
	MempoolTxAdd    = int(0x01)
	Initialized     = int(0x02)
	DownloaderStart = int(0x03)
	DownloaderEnd   = int(0x04)
)

type Event struct {
	Data interface{}
	Ack  chan<- struct{}
}

func New(data interface{}) *Event {
	return &Event{Data: data, Ack: nil}
}
