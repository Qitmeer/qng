package qitsubnet

type PublicQitSubnetAPI struct {
	q *QitSubnet
}

func NewPublicQitSubnetAPI(q *QitSubnet) *PublicQitSubnetAPI {
	return &PublicQitSubnetAPI{q}
}
