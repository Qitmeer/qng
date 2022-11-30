package model

import (
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/consensus/model"
)

// DAGTopologyManager exposes methods for querying relationships
// between blocks in the DAG
type DAGTopologyManager interface {
	Parents(stagingArea *model.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error)
	Children(stagingArea *model.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error)
	IsParentOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsChildOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsAncestorOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsAncestorOfAny(stagingArea *model.StagingArea, blockHash *hash.Hash, potentialDescendants []*hash.Hash) (bool, error)
	IsAnyAncestorOf(stagingArea *model.StagingArea, potentialAncestors []*hash.Hash, blockHash *hash.Hash) (bool, error)
	IsInSelectedParentChainOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	ChildInSelectedParentChainOf(stagingArea *model.StagingArea, lowHash, highHash *hash.Hash) (*hash.Hash, error)

	SetParents(stagingArea *model.StagingArea, blockHash *hash.Hash, parentHashes []*hash.Hash) error
}
