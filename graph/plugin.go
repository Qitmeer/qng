package graph

import (
	"fmt"
	"strconv"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/rerorero/prerogel/plugin"
)

const aggregatorName = "prerogel-graph"

var _ = (plugin.Vertex)(&vert{})
var _ = (plugin.Plugin)(&Plugin{})
var aggregators = []plugin.Aggregator{&Aggregator{}}

// vert is vertex of graph
type vert struct {
	id            string
	value         *Node
	outgoingEdges []EdgeFunction
}

func (v *vert) Compute(ctx plugin.ComputeContext) error {
	// no messages are sent in superstep 0
	if ctx.SuperStep() > 0 {
		messages := ctx.ReceivedMessages()

		if len(messages) == 0 {
			ctx.VoteToHalt()
			return ctx.PutAggregatable(aggregatorName, v.value)
		}
	}
	return ctx.PutAggregatable(aggregatorName, v.value)
}

func (v *vert) GetID() plugin.VertexID {
	return plugin.VertexID(v.id)
}

func (v *vert) GetValueAsString() string {
	return v.value.Name
}

type Aggregator struct{}

func (agg *Aggregator) Name() string {
	return aggregatorName
}

func (agg *Aggregator) Aggregate(v1 plugin.AggregatableValue, v2 plugin.AggregatableValue) (plugin.AggregatableValue, error) {
	v1n, ok1 := v1.(uint32)
	v2n, ok2 := v2.(uint32)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown value: v1=%#v, v2=%#v", v1, v2)
	}
	if v1n > v2n {
		return v1n, nil
	}
	return v2n, nil
}

func (agg *Aggregator) MarshalValue(v plugin.AggregatableValue) (*types.Any, error) {
	return plugin.ConvertUint32ToAny(v)
}

func (agg *Aggregator) UnmarshalValue(pb *types.Any) (plugin.AggregatableValue, error) {
	return plugin.ConvertAnyToUint32(pb)
}

func (agg *Aggregator) ToString(v plugin.AggregatableValue) string {
	n, ok := v.(uint32)
	if !ok {
		return fmt.Sprintf("<unknown value: %#v>", v)
	}
	return strconv.FormatUint(uint64(n), 10)
}

type Plugin struct {
	graph *MessageGraphLoader
}

// NewVertex returns a new Vertex instance
func (p *Plugin) NewVertex(id plugin.VertexID) (plugin.Vertex, error) {
	value, outgoings, err := p.graph.LoadVertex(string(id))
	if err != nil {
		return nil, err
	}
	return &vert{
		id:            string(id),
		value:         value,
		outgoingEdges: outgoings,
	}, nil
}
func (p *Plugin) NewPartitionVertices(partitionID uint64, numOfPartitions uint64, register func(v plugin.Vertex)) error {
	values, outgoings, err := p.graph.LoadPartition(partitionID, numOfPartitions)
	if err != nil {
		return err
	}

	for id, value := range values {
		register(&vert{
			id:            string(id),
			value:         value,
			outgoingEdges: outgoings[id],
		})
	}

	return nil
}

// Partition provides partition information
func (p *Plugin) Partition(vertex plugin.VertexID, numOfPartitions uint64) (uint64, error) {
	return plugin.HashPartition(vertex, numOfPartitions)
}

// MarshalMessage converts plugin.message to protobuf Any
func (p *Plugin) MarshalMessage(msg plugin.Message) (*types.Any, error) {
	return plugin.ConvertUint32ToAny(msg)
}

// UnmarshalMessage converts protobuf Any to plugin.message
func (p *Plugin) UnmarshalMessage(pb *types.Any) (plugin.Message, error) {
	return plugin.ConvertAnyToUint32(pb)
}

func combiner(destination plugin.VertexID, messages []plugin.Message) ([]plugin.Message, error) {
	// choose the highest number
	max, err := getMaxFromMessages(messages)
	if err != nil {
		return nil, err
	}
	return []plugin.Message{max}, nil
}

func getMaxFromMessages(messages []plugin.Message) (uint32, error) {
	if len(messages) == 0 {
		return 0, errors.New("expects non empty slice")
	}
	var max uint32
	for _, m := range messages {
		mv, ok := m.(uint32)
		if !ok {
			return 0, fmt.Errorf("unknown mesage type: %#v", m)
		}
		if mv > max {
			max = mv
		}
	}
	return max, nil
}

// GetCombiner returns combiner function
func (p *Plugin) GetCombiner() func(destination plugin.VertexID, messages []plugin.Message) ([]plugin.Message, error) {
	return combiner
}

// GetAggregators returns aggregators to be registered
func (p *Plugin) GetAggregators() []plugin.Aggregator {
	return aggregators
}
