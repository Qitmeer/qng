package model

type TxManager interface {
	MemPool() TxPool
	FeeEstimator() FeeEstimator
	InitDefaultFeeEstimator()
}