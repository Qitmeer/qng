package qit

type PublicQitServiceAPI struct {
	q *QitService
}

func NewPublicQitServiceAPI(q *QitService) *PublicQitServiceAPI {
	return &PublicQitServiceAPI{q}
}
