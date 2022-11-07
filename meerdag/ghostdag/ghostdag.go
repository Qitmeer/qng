package ghostdag

import (
	"github.com/Qitmeer/qng/common/hash"
	cmodel "github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/meerdag/ghostdag/model"
	"github.com/pkg/errors"
	"math/big"
	"sort"
)

// GhostDAG resolves and manages GHOSTDAG block data
type GhostDAG struct {
	databaseContext    model.DBReader
	dagTopologyManager model.DAGTopologyManager
	ghostdagDataStore  model.GHOSTDAGDataStore
	headerStore        model.BlockHeaderStore

	k           model.KType
	genesisHash *hash.Hash
}

// GHOSTDAG runs the GHOSTDAG protocol and calculates the block BlockGHOSTDAGData by the given parents.
// The function calculates MergeSetBlues by iterating over the blocks in
// the anticone of the new block selected parent (which is the parent with the
// highest blue score) and adds any block to newNode.blues if by adding
// it to MergeSetBlues these conditions will not be violated:
//
// 1) |anticone-of-candidate-block ∩ blue-set-of-newBlock| ≤ K
//
// 2) For every blue block in blue-set-of-newBlock:
//    |(anticone-of-blue-block ∩ blue-set-newBlock) ∪ {candidate-block}| ≤ K.
//    We validate this condition by maintaining a map BluesAnticoneSizes for
//    each block which holds all the blue anticone sizes that were affected by
//    the new added blue blocks.
//    So to find out what is |anticone-of-blue ∩ blue-set-of-newBlock| we just iterate in
//    the selected parent chain of the new block until we find an existing entry in
//    BluesAnticoneSizes.
//
// For further details see the article https://eprint.iacr.org/2018/104.pdf
func (gm *GhostDAG) GHOSTDAG(stagingArea *cmodel.StagingArea, blockHash *hash.Hash) error {
	newBlockData := model.NewBlockGHOSTDAGData(0, new(big.Int), nil, make([]*hash.Hash, 0), make([]*hash.Hash, 0), make(map[hash.Hash]model.KType))
	blockParents, err := gm.dagTopologyManager.Parents(stagingArea, blockHash)
	if err != nil {
		return err
	}

	isGenesis := len(blockParents) == 0
	if !isGenesis {
		selectedParent, err := gm.findSelectedParent(stagingArea, blockParents)
		if err != nil {
			return err
		}
		newBlockData.SetSelectedParent(selectedParent)
		newBlockData.AppendMergeSetBlue(selectedParent)
		newBlockData.SetBluesAnticoneSize(selectedParent, 0)
	}

	mergeSetWithoutSelectedParent, err := gm.mergeSetWithoutSelectedParent(
		stagingArea, newBlockData.SelectedParent(), blockParents)
	if err != nil {
		return err
	}

	for _, blueCandidate := range mergeSetWithoutSelectedParent {
		isBlue, candidateAnticoneSize, candidateBluesAnticoneSizes, err := gm.checkBlueCandidate(
			stagingArea, newBlockData.Clone(), blueCandidate)
		if err != nil {
			return err
		}

		if isBlue {
			// No k-cluster violation found, we can now set the candidate block as blue
			newBlockData.AppendMergeSetBlue(blueCandidate)
			newBlockData.SetBluesAnticoneSize(blueCandidate, candidateAnticoneSize)
			for blue, blueAnticoneSize := range candidateBluesAnticoneSizes {
				newBlockData.SetBluesAnticoneSize(&blue, blueAnticoneSize+1)
			}
		} else {
			newBlockData.AppendMergeSetRed(blueCandidate)
		}
	}

	if !isGenesis {
		selectedParentGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, newBlockData.SelectedParent(), false)
		if err != nil {
			return err
		}
		newBlockData.SetBlueScore(selectedParentGHOSTDAGData.BlueScore() + uint64(newBlockData.MergeSetBluesLen()))
		// We inherit the bluework from the selected parent
		newBlockData.SetBlueWork(selectedParentGHOSTDAGData.BlueWork())
		// Then we add up all the *work*(not blueWork) that all of newBlock merge set blues and selected parent did
		for _, blue := range newBlockData.MergeSetBlues() {
			// We don't count the work of the virtual genesis
			if blue.IsEqual(&model.VirtualGenesisBlockHash) {
				continue
			}

			header, err := gm.headerStore.BlockHeader(gm.databaseContext, stagingArea, blue)
			if err != nil {
				return err
			}
			newBlockData.AddBlueWork(newBlockData.BlueWork(), pow.CalcWork(header.Bits(), header.Pow().GetPowType()))
		}
	} else {
		// Genesis's blue score is defined to be 0.
		newBlockData.SetBlueScore(0)
		newBlockData.SetBlueWorkUint64(0)
	}

	gm.ghostdagDataStore.Stage(stagingArea, blockHash, newBlockData.Clone(), false)

	return nil
}

type chainBlockData struct {
	hash      *hash.Hash
	blockData *model.BlockGHOSTDAGData
}

func (gm *GhostDAG) checkBlueCandidate(stagingArea *cmodel.StagingArea, newBlockData *model.BlockGHOSTDAGData,
	blueCandidate *hash.Hash) (isBlue bool, candidateAnticoneSize model.KType,
	candidateBluesAnticoneSizes map[hash.Hash]model.KType, err error) {

	// The maximum length of node.blues can be K+1 because
	// it contains the selected parent.
	if model.KType(len(newBlockData.MergeSetBlues())) == gm.k+1 {
		return false, 0, nil, nil
	}

	candidateBluesAnticoneSizes = make(map[hash.Hash]model.KType, gm.k)

	// Iterate over all blocks in the blue set of newNode that are not in the past
	// of blueCandidate, and check for each one of them if blueCandidate potentially
	// enlarges their blue anticone to be over K, or that they enlarge the blue anticone
	// of blueCandidate to be over K.
	chainBlock := chainBlockData{
		blockData: newBlockData,
	}

	for {
		isBlue, isRed, err := gm.checkBlueCandidateWithChainBlock(stagingArea, newBlockData, chainBlock, blueCandidate,
			candidateBluesAnticoneSizes, &candidateAnticoneSize)
		if err != nil {
			return false, 0, nil, err
		}

		if isBlue {
			break
		}

		if isRed {
			return false, 0, nil, nil
		}

		selectedParentGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, chainBlock.blockData.SelectedParent(), false)
		if err != nil {
			return false, 0, nil, err
		}

		chainBlock = chainBlockData{hash: chainBlock.blockData.SelectedParent(),
			blockData: selectedParentGHOSTDAGData,
		}
	}

	return true, candidateAnticoneSize, candidateBluesAnticoneSizes, nil
}

func (gm *GhostDAG) checkBlueCandidateWithChainBlock(stagingArea *cmodel.StagingArea,
	newBlockData *model.BlockGHOSTDAGData, chainBlock chainBlockData, blueCandidate *hash.Hash,
	candidateBluesAnticoneSizes map[hash.Hash]model.KType,
	candidateAnticoneSize *model.KType) (isBlue, isRed bool, err error) {

	// If blueCandidate is in the future of chainBlock, it means
	// that all remaining blues are in the past of chainBlock and thus
	// in the past of blueCandidate. In this case we know for sure that
	// the anticone of blueCandidate will not exceed K, and we can mark
	// it as blue.
	//
	// The new block is always in the future of blueCandidate, so there's
	// no point in checking it.

	// We check if chainBlock is not the new block by checking if it has a hash.
	if chainBlock.hash != nil {
		isAncestorOfBlueCandidate, err := gm.dagTopologyManager.IsAncestorOf(stagingArea, chainBlock.hash, blueCandidate)
		if err != nil {
			return false, false, err
		}
		if isAncestorOfBlueCandidate {
			return true, false, nil
		}
	}

	for _, block := range chainBlock.blockData.MergeSetBlues() {
		// Skip blocks that exist in the past of blueCandidate.
		isAncestorOfBlueCandidate, err := gm.dagTopologyManager.IsAncestorOf(stagingArea, block, blueCandidate)
		if err != nil {
			return false, false, err
		}

		if isAncestorOfBlueCandidate {
			continue
		}

		candidateBluesAnticoneSizes[*block], err = gm.blueAnticoneSize(stagingArea, block, newBlockData)
		if err != nil {
			return false, false, err
		}
		*candidateAnticoneSize++

		if *candidateAnticoneSize > gm.k {
			// k-cluster violation: The candidate's blue anticone exceeded k
			return false, true, nil
		}

		if candidateBluesAnticoneSizes[*block] == gm.k {
			// k-cluster violation: A block in candidate's blue anticone already
			// has k blue blocks in its own anticone
			return false, true, nil
		}

		// This is a sanity check that validates that a blue
		// block's blue anticone is not already larger than K.
		if candidateBluesAnticoneSizes[*block] > gm.k {
			return false, false, errors.New("found blue anticone size larger than k")
		}
	}

	return false, false, nil
}

// blueAnticoneSize returns the blue anticone size of 'block' from the worldview of 'context'.
// Expects 'block' to be in the blue set of 'context'
func (gm *GhostDAG) blueAnticoneSize(stagingArea *cmodel.StagingArea,
	block *hash.Hash, context *model.BlockGHOSTDAGData) (model.KType, error) {

	isTrustedData := false
	for current := context; current != nil; {
		if blueAnticoneSize, ok := current.BluesAnticoneSizes()[*block]; ok {
			return blueAnticoneSize, nil
		}
		if current.SelectedParent().IsEqual(gm.genesisHash) {
			break
		}

		var err error
		current, err = gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, current.SelectedParent(), isTrustedData)
		if err != nil {
			return 0, err
		}
		if current.SelectedParent().IsEqual(&model.VirtualGenesisBlockHash) {
			isTrustedData = true
			current, err = gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, current.SelectedParent(), isTrustedData)
			if err != nil {
				return 0, err
			}
		}
	}
	return 0, errors.Errorf("block %s is not in blue set of the given context", block)
}

// compare
func (gm *GhostDAG) findSelectedParent(stagingArea *cmodel.StagingArea, parentHashes []*hash.Hash) (*hash.Hash, error) {
	var selectedParent *hash.Hash
	for _, hash := range parentHashes {
		if selectedParent == nil {
			selectedParent = hash
			continue
		}
		isHashBiggerThanSelectedParent, err := gm.less(stagingArea, selectedParent, hash)
		if err != nil {
			return nil, err
		}
		if isHashBiggerThanSelectedParent {
			selectedParent = hash
		}
	}
	return selectedParent, nil
}

func (gm *GhostDAG) FindSelectedParent(stagingArea *cmodel.StagingArea, parentHashes []*hash.Hash) (*hash.Hash, error) {
	return gm.findSelectedParent(stagingArea, parentHashes)
}

func (gm *GhostDAG) less(stagingArea *cmodel.StagingArea, blockHashA, blockHashB *hash.Hash) (bool, error) {
	chosenSelectedParent, err := gm.ChooseSelectedParent(stagingArea, blockHashA, blockHashB)
	if err != nil {
		return false, err
	}
	return chosenSelectedParent == blockHashB, nil
}

func (gm *GhostDAG) ChooseSelectedParent(stagingArea *cmodel.StagingArea, blockHashes ...*hash.Hash) (*hash.Hash, error) {
	selectedParent := blockHashes[0]
	selectedParentGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, selectedParent, false)
	if err != nil {
		return nil, err
	}
	for _, blockHash := range blockHashes {
		blockGHOSTDAGData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, blockHash, false)
		if err != nil {
			return nil, err
		}

		if gm.Less(selectedParent, selectedParentGHOSTDAGData, blockHash, blockGHOSTDAGData) {
			selectedParent = blockHash
			selectedParentGHOSTDAGData = blockGHOSTDAGData
		}
	}

	return selectedParent, nil
}

func (gm *GhostDAG) Less(blockHashA *hash.Hash, ghostdagDataA *model.BlockGHOSTDAGData,
	blockHashB *hash.Hash, ghostdagDataB *model.BlockGHOSTDAGData) bool {
	switch ghostdagDataA.BlueWork().Cmp(ghostdagDataB.BlueWork()) {
	case -1:
		return true
	case 1:
		return false
	case 0:
		return blockHashA.Less(blockHashB)
	default:
		panic("big.Int.Cmp is defined to always return -1/1/0 and nothing else")
	}
}

// mergeset
func (gm *GhostDAG) mergeSetWithoutSelectedParent(stagingArea *cmodel.StagingArea, selectedParent *hash.Hash, blockParents []*hash.Hash) ([]*hash.Hash, error) {

	mergeSetMap := make(map[hash.Hash]struct{}, gm.k)
	mergeSetSlice := make([]*hash.Hash, 0, gm.k)
	selectedParentPast := make(map[hash.Hash]struct{})
	queue := []*hash.Hash{}
	// Queueing all parents (other than the selected parent itself) for processing.
	for _, parent := range blockParents {
		if parent.IsEqual(selectedParent) {
			continue
		}
		mergeSetMap[*parent] = struct{}{}
		mergeSetSlice = append(mergeSetSlice, parent)
		queue = append(queue, parent)
	}

	for len(queue) > 0 {
		var current *hash.Hash
		current, queue = queue[0], queue[1:]
		// For each parent of the current block we check whether it is in the past of the selected parent. If not,
		// we add the it to the resulting anticone-set and queue it for further processing.
		currentParents, err := gm.dagTopologyManager.Parents(stagingArea, current)
		if err != nil {
			return nil, err
		}
		for _, parent := range currentParents {
			if _, ok := mergeSetMap[*parent]; ok {
				continue
			}

			if _, ok := selectedParentPast[*parent]; ok {
				continue
			}

			isAncestorOfSelectedParent, err := gm.dagTopologyManager.IsAncestorOf(stagingArea, parent, selectedParent)
			if err != nil {
				return nil, err
			}

			if isAncestorOfSelectedParent {
				selectedParentPast[*parent] = struct{}{}
				continue
			}

			mergeSetMap[*parent] = struct{}{}
			mergeSetSlice = append(mergeSetSlice, parent)
			queue = append(queue, parent)
		}
	}

	err := gm.sortMergeSet(stagingArea, mergeSetSlice)
	if err != nil {
		return nil, err
	}

	return mergeSetSlice, nil
}

func (gm *GhostDAG) sortMergeSet(stagingArea *cmodel.StagingArea, mergeSetSlice []*hash.Hash) error {
	var err error
	sort.Slice(mergeSetSlice, func(i, j int) bool {
		if err != nil {
			return false
		}
		isLess, lessErr := gm.less(stagingArea, mergeSetSlice[i], mergeSetSlice[j])
		if lessErr != nil {
			err = lessErr
			return false
		}
		return isLess
	})
	return err
}

// GetSortedMergeSet return the merge set sorted in a toplogical order.
func (gm *GhostDAG) GetSortedMergeSet(stagingArea *cmodel.StagingArea,
	current *hash.Hash) ([]*hash.Hash, error) {

	currentGhostdagData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, current, false)
	if err != nil {
		return nil, err
	}

	blueMergeSet := currentGhostdagData.MergeSetBlues()
	redMergeSet := currentGhostdagData.MergeSetReds()
	sortedMergeSet := make([]*hash.Hash, 0, len(blueMergeSet)+len(redMergeSet))
	// If the current block is the genesis block:
	if len(blueMergeSet) == 0 {
		return sortedMergeSet, nil
	}
	selectedParent, blueMergeSet := blueMergeSet[0], blueMergeSet[1:]
	sortedMergeSet = append(sortedMergeSet, selectedParent)
	i, j := 0, 0
	for i < len(blueMergeSet) && j < len(redMergeSet) {
		currentBlue := blueMergeSet[i]
		currentBlueGhostdagData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, currentBlue, false)
		if err != nil {
			return nil, err
		}
		currentRed := redMergeSet[j]
		currentRedGhostdagData, err := gm.ghostdagDataStore.Get(gm.databaseContext, stagingArea, currentRed, false)
		if err != nil {
			return nil, err
		}
		if gm.Less(currentBlue, currentBlueGhostdagData, currentRed, currentRedGhostdagData) {
			sortedMergeSet = append(sortedMergeSet, currentBlue)
			i++
		} else {
			sortedMergeSet = append(sortedMergeSet, currentRed)
			j++
		}
	}
	sortedMergeSet = append(sortedMergeSet, blueMergeSet[i:]...)
	sortedMergeSet = append(sortedMergeSet, redMergeSet[j:]...)

	return sortedMergeSet, nil
}

func (gm *GhostDAG) GenesisHash() *hash.Hash {
	return gm.genesisHash
}

func (gm *GhostDAG) SetGenesisHash(h *hash.Hash) {
	gm.genesisHash = h
}

// New instantiates a new GhostDAG
func New(
	databaseContext model.DBReader,
	dagTopologyManager model.DAGTopologyManager,
	ghostdagDataStore model.GHOSTDAGDataStore,
	headerStore model.BlockHeaderStore,
	k model.KType,
	genesisHash *hash.Hash) *GhostDAG {

	return &GhostDAG{
		databaseContext:    databaseContext,
		dagTopologyManager: dagTopologyManager,
		ghostdagDataStore:  ghostdagDataStore,
		headerStore:        headerStore,
		k:                  k,
		genesisHash:        genesisHash,
	}
}
