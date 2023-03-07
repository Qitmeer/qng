package consensus

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerevm/bridge"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

// CommitStates commit states
func (c *Qit) CommitStates(
	ctx context.Context,
	state *state.StateDB,
	header *types.Header,
	chain bridge.ChainContext,
) ([]*bridge.StateSyncData, error) {
	fetchStart := time.Now()
	number := header.Number.Uint64()

	_lastStateID, err := c.GenesisContractsClient.LastStateId(number - 1)
	if err != nil {
		return nil, err
	}

	to := time.Unix(int64(chain.Chain.GetHeaderByNumber(number).Time), 0)
	lastStateID := _lastStateID.Uint64()

	eventRecords := []*bridge.EventRecordWithTime{}
	log.Info(
		"Fetching state updates",
		"fromID", lastStateID+1,
		"to", to.Format(time.RFC3339))

	fetchTime := time.Since(fetchStart)
	processStart := time.Now()
	totalGas := 0 /// limit on gas for state sync per block
	chainID := ""
	stateSyncs := make([]*bridge.StateSyncData, len(eventRecords))

	var gasUsed uint64

	for _, eventRecord := range eventRecords {
		if eventRecord.ID <= lastStateID {
			continue
		}
		if err = validateEventRecord(eventRecord, number, to, lastStateID, chainID); err != nil {
			log.Error("while validating event record", "block", number, "to", to, "stateID", lastStateID, "error", err.Error())
			break
		}

		stateData := bridge.StateSyncData{
			ID:       eventRecord.ID,
			Contract: eventRecord.Contract,
			Data:     hex.EncodeToString(eventRecord.Data),
			TxHash:   eventRecord.TxHash,
		}

		stateSyncs = append(stateSyncs, &stateData)
		gasUsed, err = c.GenesisContractsClient.CommitState(eventRecord, state, header, chain)
		if err != nil {
			return nil, err
		}

		totalGas += int(gasUsed)

		lastStateID++
	}

	processTime := time.Since(processStart)

	log.Info("StateSyncData", "gas", totalGas, "number", number, "lastStateID", lastStateID, "total records", len(eventRecords), "fetch time", int(fetchTime.Milliseconds()), "process time", int(processTime.Milliseconds()))

	return stateSyncs, nil
}

func validateEventRecord(eventRecord *bridge.EventRecordWithTime, number uint64, to time.Time, lastStateID uint64, chainID string) error {
	// event id should be sequential and event.Time should lie in the range [from, to)
	if lastStateID+1 != eventRecord.ID || eventRecord.ChainID != chainID || !eventRecord.Time.Before(to) {
		return &InvalidStateReceivedError{number, lastStateID, &to, eventRecord}
	}

	return nil
}

type InvalidStateReceivedError struct {
	Number      uint64
	LastStateID uint64
	To          *time.Time
	Event       *bridge.EventRecordWithTime
}

func (e *InvalidStateReceivedError) Error() string {
	return fmt.Sprintf(
		"Received invalid event %v at block %d. Requested events until %s. Last state id was %d",
		e.Event,
		e.Number,
		e.To.Format(time.RFC3339),
		e.LastStateID,
	)
}
