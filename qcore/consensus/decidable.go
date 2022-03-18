/*
 * Copyright (c) 2017-2020 The qitmeer developers
 */

package consensus

import "github.com/Qitmeer/qng-core/common/hash"

type Decidable interface {
	ID() *hash.Hash
	Accept() error
	Reject() error
	Status() Status
}
