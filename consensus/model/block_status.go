package model

// BlockStatus
type BlockStatus byte

const (
	// StatusNone
	StatusNone BlockStatus = 0

	// StatusBadSide
	StatusBadSide BlockStatus = 1 << 0

	// StatusInvalid indicates that the block data has failed validation.
	StatusInvalid BlockStatus = 1 << 2
)

func (status BlockStatus) IsBadSide() bool {
	return status&StatusBadSide != 0
}

func (status BlockStatus) KnownInvalid() bool {
	return status&StatusInvalid != 0
}
