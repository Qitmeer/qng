package consensus

type TxType uint32

const (
	TxTypeNormal TxType = iota
	TxTypeExport TxType = 1
	TxTypeImport TxType = 2
)

type Tx struct {
	Type  TxType
	From  string
	To    string
	Value uint64
	Data  []byte
}
