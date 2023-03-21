package driver

import (
	"context"
	"time"

	"github.com/Qitmeer/qit/common"
	"github.com/Qitmeer/qit/log"

	"github.com/Qitmeer/qng/rollups/node/eth"
	"github.com/Qitmeer/qng/rollups/node/rollup"
	"github.com/Qitmeer/qng/rollups/node/rollup/derive"
	"github.com/Qitmeer/qng/rollups/node/sources"
)

type Metrics interface {
	RecordPipelineReset()
	RecordPublishingError()
	RecordDerivationError()

	RecordReceivedUnsafePayload(payload *eth.ExecutionPayload)

	RecordL1Ref(name string, ref eth.L1BlockRef)
	RecordL2Ref(name string, ref eth.L2BlockRef)

	RecordUnsafePayloadsBuffer(length uint64, memSize uint64, next eth.BlockID)

	SetDerivationIdle(idle bool)

	RecordL1ReorgDepth(d uint64)

	EngineMetrics
	SequencerMetrics
}

type L1Chain interface {
	derive.L1Fetcher
	L1BlockRefByLabel(context.Context, eth.BlockLabel) (eth.L1BlockRef, error)
}

type L2Chain interface {
	derive.Engine
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
}

type DerivationPipeline interface {
	Reset()
	Step(ctx context.Context) error
	AddUnsafePayload(payload *eth.ExecutionPayload)
	GetUnsafeQueueGap(expectedNumber uint64) (uint64, uint64)
	Finalize(ref eth.L1BlockRef)
	FinalizedL1() eth.L1BlockRef
	Finalized() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	Origin() eth.L1BlockRef
}

type L1StateIface interface {
	HandleNewL1HeadBlock(head eth.L1BlockRef)
	HandleNewL1SafeBlock(safe eth.L1BlockRef)
	HandleNewL1FinalizedBlock(finalized eth.L1BlockRef)

	L1Head() eth.L1BlockRef
	L1Safe() eth.L1BlockRef
	L1Finalized() eth.L1BlockRef
}

type SequencerIface interface {
	StartBuildingBlock(ctx context.Context) error
	CompleteBuildingBlock(ctx context.Context) (*eth.ExecutionPayload, error)
	PlanNextSequencerAction() time.Duration
	RunNextSequencerAction(ctx context.Context) (*eth.ExecutionPayload, error)
	BuildingOnto() eth.L2BlockRef
}

type Network interface {
	// PublishL2Payload is called by the driver whenever there is a new payload to publish, synchronously with the driver main loop.
	PublishL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error
}

// NewDriver composes an events handler that tracks L1 state, triggers L2 derivation, and optionally sequences new L2 blocks.
func NewDriver(driverCfg *Config, cfg *rollup.Config, l2 L2Chain, l1 L1Chain, syncClient *sources.SyncClient, network Network, log log.Logger, snapshotLog log.Logger, metrics Metrics) *Driver {
	l1State := NewL1State(log, metrics)
	sequencerConfDepth := NewConfDepth(driverCfg.SequencerConfDepth, l1State.L1Head, l1)
	findL1Origin := NewL1OriginSelector(log, cfg, sequencerConfDepth)
	verifConfDepth := NewConfDepth(driverCfg.VerifierConfDepth, l1State.L1Head, l1)
	derivationPipeline := derive.NewDerivationPipeline(log, cfg, verifConfDepth, l2, metrics)
	attrBuilder := derive.NewFetchingAttributesBuilder(cfg, l1, l2)
	engine := derivationPipeline
	meteredEngine := NewMeteredEngine(cfg, engine, metrics, log)
	sequencer := NewSequencer(log, cfg, meteredEngine, attrBuilder, findL1Origin, metrics)

	return &Driver{
		l1State:          l1State,
		derivation:       derivationPipeline,
		stateReq:         make(chan chan struct{}),
		forceReset:       make(chan chan struct{}, 10),
		startSequencer:   make(chan hashAndErrorChannel, 10),
		stopSequencer:    make(chan chan hashAndError, 10),
		config:           cfg,
		driverConfig:     driverCfg,
		done:             make(chan struct{}),
		log:              log,
		snapshotLog:      snapshotLog,
		l1:               l1,
		l2:               l2,
		sequencer:        sequencer,
		network:          network,
		metrics:          metrics,
		l1HeadSig:        make(chan eth.L1BlockRef, 10),
		l1SafeSig:        make(chan eth.L1BlockRef, 10),
		l1FinalizedSig:   make(chan eth.L1BlockRef, 10),
		unsafeL2Payloads: make(chan *eth.ExecutionPayload, 10),
		L2SyncCl:         syncClient,
	}
}
