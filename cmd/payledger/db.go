package main

const (
	// blockDbNamePrefix is the prefix for the block database name.  The
	// database type is appended to this value to form the full block
	// database name.
	blockDbNamePrefix = "blocks"
)

var (
	// DebugAddrInfoBucketName is the name of the db bucket used to house the
	// debug address info
	DebugAddrInfoBucketName = []byte("debugaddrinfo")
)
