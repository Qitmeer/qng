package derive

import (
	"bytes"
	"context"
	"io"

	"github.com/Qitmeer/qit/log"

	"github.com/Qitmeer/qng/rollups/node/eth"
)

// Channel In Reader reads a batch from the channel
// This does decompression and limits the max RLP size
// This is a pure function from the channel, but each channel (or channel fragment)
// must be tagged with an L1 inclusion block to be passed to the batch queue.

type ChannelInReader struct {
	log log.Logger

	nextBatchFn func() (BatchWithL1InclusionBlock, error)

	prev *ChannelBank
}

var _ ResetableStage = (*ChannelInReader)(nil)

// NewChannelInReader creates a ChannelInReader, which should be Reset(origin) before use.
func NewChannelInReader(log log.Logger, prev *ChannelBank) *ChannelInReader {
	return &ChannelInReader{
		log:  log,
		prev: prev,
	}
}

func (cr *ChannelInReader) Origin() eth.L1BlockRef {
	return cr.prev.Origin()
}

// TODO: Take full channel for better logging
func (cr *ChannelInReader) WriteChannel(data []byte) error {
	if f, err := BatchReader(bytes.NewBuffer(data), cr.Origin()); err == nil {
		cr.nextBatchFn = f
		return nil
	} else {
		cr.log.Error("Error creating batch reader from channel data", "err", err)
		return err
	}
}

// NextChannel forces the next read to continue with the next channel,
// resetting any decoding/decompression state to a fresh start.
func (cr *ChannelInReader) NextChannel() {
	cr.nextBatchFn = nil
}

// NextBatch pulls out the next batch from the channel if it has it.
// It returns io.EOF when it cannot make any more progress.
// It will return a temporary error if it needs to be called again to advance some internal state.
func (cr *ChannelInReader) NextBatch(ctx context.Context) (*BatchData, error) {
	if cr.nextBatchFn == nil {
		if data, err := cr.prev.NextData(ctx); err == io.EOF {
			return nil, io.EOF
		} else if err != nil {
			return nil, err
		} else {
			if err := cr.WriteChannel(data); err != nil {
				return nil, NewTemporaryError(err)
			}
		}
	}

	// TODO: can batch be non nil while err == io.EOF
	// This depends on the behavior of rlp.Stream
	batch, err := cr.nextBatchFn()
	if err == io.EOF {
		cr.NextChannel()
		return nil, NotEnoughData
	} else if err != nil {
		cr.log.Warn("failed to read batch from channel reader, skipping to next channel now", "err", err)
		cr.NextChannel()
		return nil, NotEnoughData
	}
	return batch.Batch, nil
}

func (cr *ChannelInReader) Reset(ctx context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	cr.nextBatchFn = nil
	return io.EOF
}
