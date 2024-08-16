/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
	"errors"
	"fmt"
)

// ErrorCode identifies a kind of error.
type ErrorCode int

const (
	// There are no errors by default
	ErrNone ErrorCode = iota

	// p2p stream write error
	ErrStreamWrite

	// p2p stream read error
	ErrStreamRead

	// p2p stream base error
	ErrStreamBase

	// p2p peer unknown error
	ErrPeerUnknown

	// p2p peer bad error
	ErrBadPeer

	// p2p DAG consensus error
	ErrDAGConsensus

	// p2p message error
	ErrMessage

	// Generic rule error
	ErrGeneric

	// peer connect frequent
	ErrConnectFrequent

	// Sequence error
	ErrSequence

	// revalidate error
	ErrRevalidate

	// libp2p connect
	ErrLibp2pConnect
)

var p2pErrorCodeStrings = map[ErrorCode]string{
	ErrNone:            "No error and success",
	ErrStreamWrite:     "ErrStreamWrite",
	ErrStreamRead:      "ErrStreamRead",
	ErrStreamBase:      "ErrStreamBase",
	ErrPeerUnknown:     "ErrPeerUnknown",
	ErrBadPeer:         "ErrBadPeer",
	ErrDAGConsensus:    "ErrDAGConsensus",
	ErrMessage:         "ErrMessage",
	ErrGeneric:         "ErrGeneric",
	ErrConnectFrequent: "ErrConnectFrequent",
	ErrSequence:        "ErrSequence",
	ErrRevalidate:      "ErrRevalidate",
	ErrLibp2pConnect:   "ErrLibp2pConnect",
}

func (e ErrorCode) String() string {
	if s := p2pErrorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown P2PErrorCode (%d)", int(e))
}

func (e ErrorCode) IsSuccess() bool {
	return e == ErrNone
}

func (e ErrorCode) IsStream() bool {
	return e == ErrStreamRead || e == ErrStreamWrite || e == ErrStreamBase
}

func (e ErrorCode) IsDAGConsensus() bool {
	return e == ErrDAGConsensus
}

type Error struct {
	Code  ErrorCode
	Error error
}

func (e *Error) String() string {
	if e.Error == nil {
		return e.Code.String()
	}
	return fmt.Sprintf("%s, %s", e.Code.String(), e.Error.Error())
}

func (e *Error) Add(err string) {
	if e.Error == nil {
		e.Error = fmt.Errorf("%s", err)
		return
	}
	e.Error = fmt.Errorf("%s, %s", e.Error.Error(), err)
}

func (e *Error) AddError(err error) {
	if e.Error == nil {
		e.Error = errors.New(err.Error())
		return
	}
	e.Error = fmt.Errorf("%s, %s", e.Error.Error(), err.Error())
}

func (e *Error) ToError() error {
	return fmt.Errorf("%s", e.String())
}

func NewError(code ErrorCode, e error) *Error {
	return &Error{code, e}
}

func NewErrorStr(code ErrorCode, e string) *Error {
	return &Error{code, fmt.Errorf("%s", e)}
}

func NewSuccess() *Error {
	return &Error{Code: ErrNone}
}
