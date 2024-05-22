package model

type P2PService interface {
	IsCurrent() bool
	IsNearlySynced() bool
}
