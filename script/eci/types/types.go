package types

type EciType int

const (
	ECI_TYPE_ALIYUN = iota
)

func (this *EciType) String() string {
	switch *this {
	case ECI_TYPE_ALIYUN:
		return "AliYun"
	default:
		return "unknown"
	}
}

func GetEciType(typeName string) EciType {
	switch typeName {
	case "Aliyun":
		return ECI_TYPE_ALIYUN
	default:
		return -1
	}
}
