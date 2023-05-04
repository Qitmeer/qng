package model

// TxStatus
type TxStatus byte

const (
	TxStatusNone TxStatus = 0
	TxStatusValid TxStatus = 1
	TxStatusInvalid TxStatus = 2
	TxStatusDuplicate TxStatus = 3 // Whether duplicate tx.
)

func (status TxStatus) Invalid() bool {
	return status==TxStatusInvalid
}

func (status TxStatus) Valid() bool {
	return status==TxStatusValid
}

func (status TxStatus) Duplicate() bool {
	return status==TxStatusDuplicate
}

func (status TxStatus) String() string {
	switch status {
	case TxStatusValid:
		return "TxStatusValid"
	case TxStatusInvalid:
		return "TxStatusInvalid"
	case TxStatusDuplicate:
		return "TxStatusDuplicate"
	}
	return "TxStatusNone"
}