package common

import "github.com/Qitmeer/qng/common/hash"

type Decidable interface {
	ID() *hash.Hash
	Accept() error
	Reject() error
	Status() Status
}
