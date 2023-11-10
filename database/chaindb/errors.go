package chaindb

import "errors"

var (
	ErrDBClosed = errors.New("Database is closed")
	ErrDBAbsent = errors.New("Database is absent")
)
