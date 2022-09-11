package acct

type BalanceInfoResult struct {
	CoinId  string       `json:"coinid"`
	Balance int64        `json:"balance"`
	UTXOs   []UTXOResult `json:"utxos,omitempty"`
}

type UTXOResult struct {
	Type      string `json:"type"`
	Amount    uint64 `json:"amount"`
	PreTxHash string `json:"txid"`
	PreOutIdx uint32 `json:"idx"`
	Status    string `json:"status"`
}
