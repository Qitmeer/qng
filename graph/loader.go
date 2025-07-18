package graph

import (
	"fmt"

	"github.com/rerorero/prerogel/plugin"
)

type MessageGraphLoader struct {
	*MessageGraph
}

func (l *MessageGraphLoader) LoadVertex(id string) (*Node, []EdgeFunction, error) {
	val, ok := l.nodes[id]
	if !ok {
		return nil, nil, fmt.Errorf("id %s doesn't exist", id)
	}

	return &val, l.edgeOf(id), nil
}

func (l *MessageGraphLoader) LoadPartition(partitionID uint64, numOfPartitions uint64) (map[string]*Node, map[string][]EdgeFunction, error) {
	vertex := make(map[string]*Node)
	outgoings := make(map[string][]EdgeFunction)

	for id, v := range l.nodes {
		// this is so bad implementation
		partition, err := plugin.HashPartition(plugin.VertexID(id), numOfPartitions)
		if err != nil {
			return nil, nil, err
		}
		if partition == partitionID {
			vertex[id] = &v
			outgoings[id] = l.edgeOf(id)
		}
	}

	return vertex, outgoings, nil
}

func (l *MessageGraphLoader) edgeOf(id string) []EdgeFunction {
	var edges []EdgeFunction
	for _, e := range l.edges {
		if e.From == id {
			edges = append(edges, e.To)
		}
	}
	return edges
}
