/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package acct

import (
	"encoding/binary"
)

var (
	// ByteOrder is the preferred byte order used for serializing numeric
	// fields for storage in the database.
	ByteOrder = binary.LittleEndian

	// BalanceBucketName is the name of the db bucket used to house to
	// Address -> Balance
	BalanceBucketName = []byte("acctbalance")

	// InfoBucketName is the name of the db bucket used to house to
	// account info
	InfoBucketName = []byte("acctinfo")
)
