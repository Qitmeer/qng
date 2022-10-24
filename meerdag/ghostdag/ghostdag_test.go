package ghostdag

import (
	"encoding/json"
	"errors"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/meerdag/ghostdag/model"
	"github.com/Qitmeer/qng/params"
	"math/big"
	"os"
	"reflect"
	"testing"
)

type block struct {
	ID             string   `json:"ID"`
	Score          uint64   `json:"ExpectedScore"`
	SelectedParent string   `json:"ExpectedSelectedParent"`
	MergeSetReds   []string `json:"ExpectedReds"`
	MergeSetBlues  []string `json:"ExpectedBlues"`
	Parents        []string `json:"Parents"`
}

type testDag struct {
	K                    model.KType `json:"K"`
	GenesisID            string      `json:"GenesisID"`
	ExpectedMergeSetReds []string    `json:"ExpectedReds"`
	Blocks               []block     `json:"Blocks"`
}

func TestGHOSTDAG(t *testing.T) {
	dagTopology := &TestDAGTopologyManage{
		parentsMap: make(map[hash.Hash][]*hash.Hash),
	}

	ghostdagDataStore := &TestGHOSTDAGDataStore{
		dagMap: make(map[hash.Hash]*model.BlockGHOSTDAGData),
	}

	blockHeadersStore := &TestBlockHeadersStore{
		dagMap: make(map[hash.Hash]model.BlockHeader),
	}

	blockGHOSTDAGDataGenesis := model.NewBlockGHOSTDAGData(0, new(big.Int), nil, nil, nil, nil)
	genesisHeader := params.PrivNetParam.GenesisBlock.Header
	genesisWork := pow.CalcWork(genesisHeader.Difficulty, genesisHeader.Pow.GetPowType())

	path := "./test_data.json"
	jsonFile, err := os.Open(path)
	if err != nil {
		t.Fatalf("TestGHOSTDAG : failed opening the json file: %v", err)
	}
	defer jsonFile.Close()
	var test testDag
	decoder := json.NewDecoder(jsonFile)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&test)
	if err != nil {
		t.Fatalf("TestGHOSTDAG:failed decoding json: %v", err)
	}

	genesisHash := *StringToHash(test.GenesisID)

	dagTopology.parentsMap[genesisHash] = nil

	ghostdagDataStore.dagMap[genesisHash] = blockGHOSTDAGDataGenesis
	blockHeadersStore.dagMap[genesisHash] = NewBlockHeader(genesisHeader.Difficulty, genesisHeader.Pow)

	g := New(nil, dagTopology, ghostdagDataStore, blockHeadersStore, test.K, &genesisHash)

	for _, testBlockData := range test.Blocks {
		blockID := StringToHash(testBlockData.ID)
		dagTopology.parentsMap[*blockID] = StringToHashSlice(testBlockData.Parents)
		blockHeadersStore.dagMap[*blockID] = NewBlockHeader(genesisHeader.Difficulty, genesisHeader.Pow)

		err = g.GHOSTDAG(nil, blockID)
		if err != nil {
			t.Fatal(err)
		}
		ghostdagData, err := ghostdagDataStore.Get(nil, nil, blockID, false)
		if err != nil {
			t.Fatal(err)
		}

		// because the difficulty is constant and equal to genesis the work should be blueScore*genesisWork.
		expectedWork := new(big.Int).Mul(genesisWork, new(big.Int).SetUint64(testBlockData.Score))
		if expectedWork.Cmp(ghostdagData.BlueWork()) != 0 {
			t.Fatalf("\nTEST FAILED:\nBlock: %s, \nError: expected blue work %d but got %d.", testBlockData.ID, expectedWork, ghostdagData.BlueWork())
		}
		if testBlockData.Score != (ghostdagData.BlueScore()) {
			t.Fatalf("\nTEST FAILED:\nBlock: %s, \nError: expected blue score %d but got %d.",
				testBlockData.ID, testBlockData.Score, ghostdagData.BlueScore())
		}

		if !StringToHash(testBlockData.SelectedParent).IsEqual(ghostdagData.SelectedParent()) {
			t.Fatalf("\nTEST FAILED:\nBlock: %s, \nError: expected selected parent %v but got %s.",
				testBlockData.ID, testBlockData.SelectedParent, ghostdagData.SelectedParent())
		}

		if !reflect.DeepEqual(StringToHashSlice(testBlockData.MergeSetBlues), ghostdagData.MergeSetBlues()) {
			t.Fatalf("\nTEST FAILED:\nBlock: %s, \nError: expected merge set blues %v but got %v.",
				testBlockData.ID, testBlockData.MergeSetBlues, hashesToStrings(ghostdagData.MergeSetBlues()))
		}

		if !reflect.DeepEqual(StringToHashSlice(testBlockData.MergeSetReds), ghostdagData.MergeSetReds()) {
			t.Fatalf("\nTEST FAILED:\nBlock: %s, \nError: expected merge set reds %v but got %v.",
				testBlockData.ID, testBlockData.MergeSetReds, hashesToStrings(ghostdagData.MergeSetReds()))
		}
	}
}

func hashesToStrings(arr []*hash.Hash) []string {
	var strArr = make([]string, len(arr))
	for i, h := range arr {
		strArr[i] = string(h.Bytes())
	}
	return strArr
}

func StringToHash(strID string) *hash.Hash {
	var genesisHashArray [hash.HashSize]byte
	copy(genesisHashArray[:], strID)
	h := hash.MustBytesToHash(genesisHashArray[:])
	return &h
}

func StringToHashSlice(stringIDArr []string) []*hash.Hash {
	domainHashArr := make([]*hash.Hash, len(stringIDArr))
	for i, strID := range stringIDArr {
		domainHashArr[i] = StringToHash(strID)
	}
	return domainHashArr
}

// blockHeadersStore
type TestBlockHeadersStore struct {
	dagMap map[hash.Hash]model.BlockHeader
}

func (b *TestBlockHeadersStore) Discard() { panic("unimplemented") }

func (b *TestBlockHeadersStore) Commit(_ model.DBTransaction) error { panic("unimplemented") }

func (b *TestBlockHeadersStore) Stage(stagingArea *model.StagingArea, blockHash *hash.Hash, blockHeader model.BlockHeader) {
	b.dagMap[*blockHash] = blockHeader
}

func (b *TestBlockHeadersStore) IsStaged(*model.StagingArea) bool { panic("unimplemented") }

func (b *TestBlockHeadersStore) BlockHeader(dbContext model.DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash) (model.BlockHeader, error) {
	header, ok := b.dagMap[*blockHash]
	if ok {
		return header, nil
	}
	return nil, errors.New("Header isn't in the store")
}

func (b *TestBlockHeadersStore) HasBlockHeader(dbContext model.DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash) (bool, error) {
	_, ok := b.dagMap[*blockHash]
	return ok, nil
}

func (b *TestBlockHeadersStore) BlockHeaders(dbContext model.DBReader, stagingArea *model.StagingArea, blockHashes []*hash.Hash) ([]model.BlockHeader, error) {
	res := make([]model.BlockHeader, 0, len(blockHashes))
	for _, hash := range blockHashes {
		header, err := b.BlockHeader(nil, nil, hash)
		if err != nil {
			return nil, err
		}
		res = append(res, header)
	}
	return res, nil
}

func (b *TestBlockHeadersStore) Delete(stagingArea *model.StagingArea, blockHash *hash.Hash) {
	delete(b.dagMap, *blockHash)
}

func (b *TestBlockHeadersStore) Count(*model.StagingArea) uint64 {
	return uint64(len(b.dagMap))
}

// TestGHOSTDAGDataStore
type TestGHOSTDAGDataStore struct {
	dagMap map[hash.Hash]*model.BlockGHOSTDAGData
}

func (ds *TestGHOSTDAGDataStore) Stage(stagingArea *model.StagingArea, blockHash *hash.Hash, blockGHOSTDAGData *model.BlockGHOSTDAGData, isTrustedData bool) {
	ds.dagMap[*blockHash] = blockGHOSTDAGData
}

func (ds *TestGHOSTDAGDataStore) IsStaged(*model.StagingArea) bool {
	panic("implement me")
}

func (ds *TestGHOSTDAGDataStore) Commit() error {
	panic("implement me")
}

func (ds *TestGHOSTDAGDataStore) Get(dbContext model.DBReader, stagingArea *model.StagingArea, blockHash *hash.Hash, isTrustedData bool) (*model.BlockGHOSTDAGData, error) {
	v, ok := ds.dagMap[*blockHash]
	if ok {
		return v, nil
	}
	return nil, nil
}

func (ds *TestGHOSTDAGDataStore) UnstageAll(stagingArea *model.StagingArea) {
	panic("implement me")
}

// TestDAGTopologyManage
type TestDAGTopologyManage struct {
	parentsMap map[hash.Hash][]*hash.Hash
}

func (dt *TestDAGTopologyManage) ChildInSelectedParentChainOf(stagingArea *model.StagingArea, lowHash, highHash *hash.Hash) (*hash.Hash, error) {
	panic("implement me")
}

func (dt *TestDAGTopologyManage) Tips() ([]*hash.Hash, error) {
	panic("implement me")
}

func (dt *TestDAGTopologyManage) AddTip(tipHash *hash.Hash) error {
	panic("implement me")
}

func (dt *TestDAGTopologyManage) Parents(stagingArea *model.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error) {
	v, ok := dt.parentsMap[*blockHash]
	if !ok {
		return []*hash.Hash{}, nil
	}

	return v, nil
}

func (dt *TestDAGTopologyManage) Children(stagingArea *model.StagingArea, blockHash *hash.Hash) ([]*hash.Hash, error) {
	panic("unimplemented")
}

func (dt *TestDAGTopologyManage) IsParentOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	panic("unimplemented")
}

func (dt *TestDAGTopologyManage) IsChildOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	panic("unimplemented")
}

func (dt *TestDAGTopologyManage) IsAncestorOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	blockBParents, isOk := dt.parentsMap[*blockHashB]
	if !isOk {
		return false, nil
	}

	for _, parentOfB := range blockBParents {
		if parentOfB.IsEqual(blockHashA) {
			return true, nil
		}
	}

	for _, parentOfB := range blockBParents {
		isAncestorOf, err := dt.IsAncestorOf(stagingArea, blockHashA, parentOfB)
		if err != nil {
			return false, err
		}
		if isAncestorOf {
			return true, nil
		}
	}
	return false, nil

}

func (dt *TestDAGTopologyManage) IsAncestorOfAny(stagingArea *model.StagingArea, blockHash *hash.Hash, potentialDescendants []*hash.Hash) (bool, error) {
	panic("unimplemented")
}
func (dt *TestDAGTopologyManage) IsAnyAncestorOf(*model.StagingArea, []*hash.Hash, *hash.Hash) (bool, error) {
	panic("unimplemented")
}
func (dt *TestDAGTopologyManage) IsInSelectedParentChainOf(stagingArea *model.StagingArea, blockHashA *hash.Hash, blockHashB *hash.Hash) (bool, error) {
	panic("unimplemented")
}

func (dt *TestDAGTopologyManage) SetParents(stagingArea *model.StagingArea, blockHash *hash.Hash, parentHashes []*hash.Hash) error {
	panic("unimplemented")
}
