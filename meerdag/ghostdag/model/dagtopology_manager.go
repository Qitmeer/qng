package model

import "github.com/Qitmeer/qng/common/hash"

// DAGTopologyManager exposes methods for querying relationships
// between blocks in the DAG
type DAGTopologyManager interface {
	Parents(stagingArea *StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error)
	Children(stagingArea *StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error)
	IsParentOf(stagingArea *StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsChildOf(stagingArea *StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsAncestorOf(stagingArea *StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	IsAncestorOfAny(stagingArea *StagingArea, blockHash *hash.Hash, potentialDescendants []*hash.Hash) (bool, error)
	IsAnyAncestorOf(stagingArea *StagingArea, potentialAncestors []*hash.Hash, blockHash *hash.Hash) (bool, error)
	IsInSelectedParentChainOf(stagingArea *StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error)
	ChildInSelectedParentChainOf(stagingArea *StagingArea, lowHash, highHash *hash.Hash) (*hash.Hash, error)

	SetParents(stagingArea *StagingArea, blockHash *hash.Hash, parentHashes []*hash.Hash) error
}
