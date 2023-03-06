package hotwallet

type PublicWalletServiceAPI struct {
	q *WalletService
}

func NewPublicWalletServiceAPI(q *WalletService) *PublicWalletServiceAPI {
	return &PublicWalletServiceAPI{q}
}

func (api *PublicWalletServiceAPI) GetWalletNodeInfo() (interface{}, error) {
	return nil, nil
}

type WalletInfo struct {
}
