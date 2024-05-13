package mining

import (
	"time"
)

var TargetAllowSubmitHandleDuration = 1500 * time.Millisecond // 1.5s

const MAX_ADJUST_BLOCKSIZE = 1024 * 1024 // 1MB

const MAX_ADJUSTMENT = 2

// dynamic calculate the max allow block size
// The larger the block, the longer submission takes.
// It is simplely assumed here that the correlation is linear, submitHandleTime = k*blocksize
// So to achieve the ideal submission time we need to be adaptive, blocksize /= submitchangerate
// This adjustment does not belong to the consensus,only for node optimization
func (policy *Policy) CalcMaxBlockSize(lastSubmitHandleAvgDuration time.Duration) {
	submitchangerate := float64(lastSubmitHandleAvgDuration) / float64(TargetAllowSubmitHandleDuration)
	if submitchangerate > MAX_ADJUSTMENT {
		submitchangerate = MAX_ADJUSTMENT
	}
	if submitchangerate < float64(1)/float64(MAX_ADJUSTMENT) {
		submitchangerate = float64(1) / float64(MAX_ADJUSTMENT)
	}
	maxsize := float64(policy.BlockMaxSize) / submitchangerate

	policy.BlockMaxSize = uint32(maxsize)

	// safe adjustment
	if policy.BlockMaxSize > MAX_ADJUST_BLOCKSIZE {
		policy.BlockMaxSize = MAX_ADJUST_BLOCKSIZE
	}
	if policy.BlockMaxSize < policy.BlockMinSize {
		policy.BlockMaxSize = policy.BlockMinSize
	}
}
