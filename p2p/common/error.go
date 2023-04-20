/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package common

import (
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

type Error struct {
	Code  ErrorCode
	Error error
}

func (e *Error) String() string {
	return fmt.Sprintf("%s, %s", e.Code.String(), e.Error.Error())
}

func (e *Error) Add(err string) {
	e.Error = fmt.Errorf("%s, %s", e.Error.Error(), err)
}

func NewError(code ErrorCode, e error) *Error {
	return &Error{code, e}
}

func NewErrorStr(code ErrorCode, e string) *Error {
	return &Error{code, fmt.Errorf(e)}
}
