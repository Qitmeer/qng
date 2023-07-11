package chaindb

import "errors"

var (
	ErrDBClosed = errors.New("Database is closed")
)
