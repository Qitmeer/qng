package test

import (
	"encoding/json"
	"fmt"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/common/roughtime"
	"github.com/Qitmeer/qng/common/system"
	"github.com/Qitmeer/qng/config"
	"github.com/Qitmeer/qng/consensus/model"
	"github.com/Qitmeer/qng/core/types"
	"github.com/Qitmeer/qng/core/types/pow"
	"github.com/Qitmeer/qng/database"
	l "github.com/Qitmeer/qng/log"
	"github.com/Qitmeer/qng/meerdag"
	"github.com/Qitmeer/qng/params"
	"github.com/Qitmeer/qng/services/common"
	"io"
	"math/rand"
	"os"
	"time"
)

// Structure of blocks data
type TestBlocksData struct {
	Tag     string   `json:"tag"`
	Parents []string `json:"parents"`
}

// Test input and output structure
type TestInOutData struct {
	Input  string   `json:"in"`
	Output []string `json:"out"`
}

// Test input and output structure2
type TestInOutData2 struct {
	Input  string `json:"in"`
	Output int    `json:"out"`
}

type TestInOutData3 struct {
	Input  []string `json:"in"`
	Output bool     `json:"out"`
}

// Structure of test data
type TestData struct {
	PH_Fig1Blocks      []TestBlocksData `json:"PH_fig1-blocks"`
	PH_Fig2Blocks      []TestBlocksData `json:"PH_fig2-blocks"`
	PH_Fig4Blocks      []TestBlocksData `json:"PH_fig4-blocks"`
	PH_GetFutureSet    TestInOutData
	PH_GetAnticone     TestInOutData
	PH_BlueSetFig1     TestInOutData
	PH_BlueSetFig2     TestInOutData
	PH_BlueSetFig4     TestInOutData
	PH_OrderFig1       TestInOutData
	PH_OrderFig2       TestInOutData
	PH_OrderFig4       TestInOutData
	PH_IsOnMainChain   TestInOutData
	PH_GetLayer        TestInOutData
	CO_Blocks          []TestBlocksData
	CO_GetMainChain    TestInOutData
	CO_GetOrder        TestInOutData
	SP_Blocks          []TestBlocksData
	PH_LocateBlocks    TestInOutData
	PH_LocateMaxBlocks TestInOutData
	CP_Blocks          []TestBlocksData
	PH_MPConcurrency   TestInOutData2
	PH_BConcurrency    TestInOutData2
	PH_MainChainTip    []TestInOutData3
	GD_BlueSetFig1     TestInOutData
	GD_BlueSetFig2     TestInOutData
	GD_BlueSetFig4     TestInOutData
	GD_OrderFig1       TestInOutData
	GD_OrderFig2       TestInOutData
	GD_OrderFig4       TestInOutData
}

// Load some data that phantom test need,it can use to build the dag ;This is the
// raw input data.
func loadTestData(fileName string, testData *TestData) error {
	if len(fileName) == 0 {
		return fmt.Errorf("file name error")
	}

	var f *os.File
	var err error

	f, err = os.Open(fileName)
	if err != nil {
		return err
	}

	defer func() {
		cErr := f.Close()
		if err == nil {
			err = cErr
		}
	}()
	//
	err = json.NewDecoder(f).Decode(testData)
	return err
}

// DAG block data
type TestBlock struct {
	block *types.SerializedBlock
}

// Return the hash
func (tb *TestBlock) GetHash() *hash.Hash {
	return tb.block.Hash()
}

// Get all parents set,the dag block has more than one parent
func (tb *TestBlock) GetParents() []*hash.Hash {
	return tb.block.Block().Parents
}

func (tb *TestBlock) GetMainParent() *hash.Hash {
	return tb.block.Block().Parents[0]
}

func (tb *TestBlock) GetTimestamp() int64 {
	return tb.block.Block().Header.Timestamp.Unix()
}

// Acquire the weight of block
func (tb *TestBlock) GetWeight() uint64 {
	return 1
}

func (tb *TestBlock) GetPriority() int {
	return meerdag.MaxPriority
}

// This is the interface for Block DAG,can use to call public function.
var bd *meerdag.MeerDAG

var randTool *rand.Rand = rand.New(rand.NewSource(roughtime.Now().UnixNano()))

// It contains all of test data. Convenient for you to use different input data
// and output data.
var testData *TestData

// This is the test data file name
var testDataFilePath string = "./testData.json"

var tbMap map[string]meerdag.IBlock

func InitBlockDAG(dagType string, graph string) meerdag.ConsensusAlgorithm {
	output := io.Writer(os.Stdout)
	glogger := l.NewGlogHandler(l.StreamHandler(output, l.TerminalFormat(false)))
	glogger.Verbosity(l.LvlError)
	l.Root().SetHandler(glogger)
	blockdaglogger := l.New(l.Ctx{"module": "blockdag"})
	meerdag.UseLogger(blockdaglogger)
	l.PrintOrigins(true)
	params.ActiveNetParams = &params.PrivNetParam

	testData = &TestData{}
	err := loadTestData(testDataFilePath, testData)
	if err != nil {
		return nil
	}
	var tbd []TestBlocksData
	if graph == "PH_fig1-blocks" {
		tbd = testData.PH_Fig1Blocks
	} else if graph == "PH_fig2-blocks" {
		tbd = testData.PH_Fig2Blocks
	} else if graph == "PH_fig4-blocks" {
		tbd = testData.PH_Fig4Blocks
	} else if graph == "CO_Blocks" {
		tbd = testData.CO_Blocks
	} else if graph == "SP_Blocks" {
		tbd = testData.SP_Blocks
	} else if graph == "CP_Blocks" {
		tbd = testData.CP_Blocks
	} else {
		return nil
	}
	blen := len(tbd)
	if blen < 2 {
		return nil
	}
	cfg := common.DefaultConfig(os.TempDir())
	db, err := loadBlockDB(cfg)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	bd = meerdag.New(dagType, -1, db, nil, meerdag.CreateMockBlockState, meerdag.CreateMockBlockStateFromBytes)
	instance := bd.GetInstance()
	tbMap = map[string]meerdag.IBlock{}
	for i := 0; i < blen; i++ {
		parents := []*hash.Hash{}
		for _, parent := range tbd[i].Parents {
			parents = append(parents, tbMap[parent].GetHash())
		}
		_, err := buildBlock(tbd[i].Tag, parents)
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}

	return instance
}

func buildBlock(tag string, parents []*hash.Hash) (*TestBlock, error) {
	block, ib, err := addBlock(tag, parents)
	if err != nil {
		return nil, err
	}
	err = commitBlock(tag, block, ib)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func addBlock(tag string, parents []*hash.Hash) (*TestBlock, meerdag.IBlock, error) {
	ps := parents
	if len(parents) > 1 {
		_, ps = bd.GetMainParentAndList(parents)
	}

	b := &types.Block{
		Header: types.BlockHeader{
			Pow:        pow.GetInstance(pow.MEERXKECCAKV1, 0, []byte{}),
			Timestamp:  time.Unix(int64(len(tbMap)), 0),
			Difficulty: uint32(len(tbMap)),
		},
		Parents:      ps,
		Transactions: []*types.Transaction{},
	}
	block := &TestBlock{block: types.NewBlock(b)}

	_, _, ib, _ := bd.AddBlock(block)
	if ib != nil {
		return block, ib, nil
	} else {
		return nil, nil, fmt.Errorf("Error: %s\n", tag)
	}
}

func commitBlock(tag string, block *TestBlock, ib meerdag.IBlock) error {
	tbMap[tag] = ib
	err := bd.Commit()
	if err != nil {
		return err
	}
	err = storeBlock(block)
	if err != nil {
		return err
	}
	err = dbPutTotal(bd.GetBlockTotal())
	if err != nil {
		return err
	}
	return nil
}

func getBlockTag(id uint) string {
	for k, v := range tbMap {
		if v.GetID() == id {
			return k
		}
	}
	return ""
}

func changeToIDList(list []string) []uint {
	length := len(list)
	if length == 0 {
		return nil
	}
	result := []uint{}
	for i := 0; i < length; i++ {
		result = append(result, tbMap[list[i]].GetID())
	}
	return result
}

func processResult(calRet interface{}, theory []uint) bool {

	var ret bool = true
	switch calRet.(type) {
	case []uint:
		result := calRet.([]uint)
		rLen := len(result)

		if rLen != len(theory) {
			ret = false
		}
		for i := 0; i < rLen; i++ {
			if result[i] != theory[i] {
				ret = false
				break
			}
		}
	case *meerdag.IdSet:
		result := calRet.(*meerdag.IdSet)
		okResult := meerdag.NewIdSet()
		okResult.AddList(theory)
		if !result.IsEqual(okResult) {
			ret = false
		}
	}

	if ret {
		fmt.Println("Congratulations，The result of the function is completely correct！！！")
	} else {
		fmt.Println("Failed，The result of the operation of a function is incompatible with the expectation！！！")
	}
	return ret
}

func printBlockChainTag(list []uint) {
	var result string
	for i := 0; i < len(list); i++ {
		name := getBlockTag(list[i])
		if i == 0 {
			result += name
		} else {
			result += fmt.Sprintf("-->%s", name)
		}
	}
	fmt.Println(result)
}

func printBlockSetTag(set *meerdag.IdSet) {
	var result string = "["
	isFirst := true
	for k := range set.GetMap() {
		name := getBlockTag(k)
		if isFirst {
			result += name
			isFirst = false
		} else {
			result += fmt.Sprintf(",%s", name)
		}

	}
	result += "]"
	fmt.Println(result)
}

func reverseBlockList(s []uint) []uint {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func loadBlockDB(cfg *config.Config) (model.DataBase, error) {
	ccfg := *cfg
	ccfg.Cleanup = true
	database.Cleanup(&ccfg)
	db, err := database.New(cfg, system.InterruptListener())
	if err != nil {
		return nil, err
	}
	return db, db.Init()
}

func getBlocksByTag(tags []string) []*hash.Hash {
	result := []*hash.Hash{}
	for _, v := range tags {
		ib, ok := tbMap[v]
		if !ok {
			continue
		}
		result = append(result, ib.GetHash())
	}
	return result
}

func exit() {
	database.Cleanup(common.DefaultConfig(os.TempDir()))
}

func storeBlock(block *TestBlock) error {
	return bd.DB().PutBlock(block.block)
}

func fetchBlock(h *hash.Hash) (*TestBlock, error) {
	tb := &TestBlock{}
	block, err := bd.DB().GetBlock(h)
	if err != nil {
		return nil, err
	}
	tb.block = block
	return tb, nil
}

func dbPutTotal(total uint) error {
	var serializedTotal [4]byte
	meerdag.ByteOrder.PutUint32(serializedTotal[:], uint32(total))

	return bd.DB().Put([]byte("blocktotal"), serializedTotal[:])
}

func dbGetTotal() (uint32, error) {
	total := uint32(0)
	serializedTotal, err := bd.DB().Get([]byte("blocktotal"))
	if err != nil {
		return total, err
	}
	if serializedTotal == nil {
		return total, fmt.Errorf("No data")
	}
	total = meerdag.ByteOrder.Uint32(serializedTotal)
	return total, nil
}

func dbGetGenesis() (*hash.Hash, error) {
	block := meerdag.Block{}
	ib := &meerdag.PhantomBlock{Block: &block}
	err := meerdag.DBGetDAGBlock(bd.DB(), ib)
	if err != nil {
		return nil, err
	}
	return ib.GetHash(), nil
}
