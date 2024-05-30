package acct

type BalanceInfoResult struct {
	CoinId       string       `json:"coinid"`
	Balance      int64        `json:"balance"`
	TotalBalance uint64       `json:"totalbalance,omitempty"`
	UTXOs        []UTXOResult `json:"utxos,omitempty"`
}

type UTXOResult struct {
	Type      string `json:"type"`
	Amount    uint64 `json:"amount"`
	PreTxHash string `json:"txid"`
	PreOutIdx uint32 `json:"idx"`
	Status    string `json:"status"`
}

type ValidUTXOsResult struct {
	Amount uint64       `json:"amount"`
	Num    int          `json:"total"`
	UTXOs  []UTXOResult `json:"utxos,omitempty"`
}

type AcctDebugInfo struct {
	Total       uint32 `json:"total"`
	Watcher     uint32 `json:"watcher"`
	UtxoWatcher uint32 `json:"utxowatcher"`
}
