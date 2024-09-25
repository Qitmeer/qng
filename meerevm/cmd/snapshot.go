// Copyright (c) 2017-2018 The qitmeer developers

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Qitmeer/qng/config"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"math"
	"os"
	"strconv"
	"time"

	qcommon "github.com/Qitmeer/qng/meerevm/common"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	cli "github.com/urfave/cli/v2"
)

var (
	snapshotCommand = &cli.Command{
		Name:        "snapshot",
		Usage:       "A set of commands based on the snapshot",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:      "prune-state",
				Usage:     "Prune stale ethereum state data based on the snapshot",
				ArgsUsage: "<root>",
				Action:    pruneState,
				Flags: qcommon.Merge([]cli.Flag{
					utils.BloomFilterSizeFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot prune-state <state-root>
will prune historical state data with the help of the state snapshot.
All trie nodes and contract codes that do not belong to the specified
version state will be deleted from the database. After pruning, only
two version states are available: genesis and the specific one.

The default pruning target is the HEAD-127 state.

WARNING: it's only supported in hash mode(--state.scheme=hash)".
`,
			},
			{
				Name:      "verify-state",
				Usage:     "Recalculate state hash based on the snapshot for verification",
				ArgsUsage: "<root>",
				Action:    verifyState,
				Flags:     qcommon.Merge(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot verify-state <state-root>
will traverse the whole accounts and storages set based on the specified
snapshot and recalculate the root hash of state for verification.
In other words, this command does the snapshot to trie conversion.
`,
			},
			{
				Name:      "check-dangling-storage",
				Usage:     "Check that there is no 'dangling' snap storage",
				ArgsUsage: "<root>",
				Action:    checkDanglingStorage,
				Flags:     qcommon.Merge(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot check-dangling-storage <state-root> traverses the snap storage 
data, and verifies that all snapshot storage data has a corresponding account. 
`,
			},
			{
				Name:      "inspect-account",
				Usage:     "Check all snapshot layers for the a specific account",
				ArgsUsage: "<address | hash>",
				Action:    checkAccount,
				Flags:     qcommon.Merge(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot inspect-account <address | hash> checks all snapshot layers and prints out
information about the specified address. 
`,
			},
			{
				Name:      "traverse-state",
				Usage:     "Traverse the state with given root hash and perform quick verification",
				ArgsUsage: "<root>",
				Action:    traverseState,
				Flags:     qcommon.Merge(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-state <state-root>
will traverse the whole state from the given state root and will abort if any
referenced trie node or contract code is missing. This command can be used for
state integrity verification. The default checking target is the HEAD state.

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "traverse-rawstate",
				Usage:     "Traverse the state with given root hash and perform detailed verification",
				ArgsUsage: "<root>",
				Action:    traverseRawState,
				Flags: qcommon.Merge([]cli.Flag{
					utils.StateSchemeFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-rawstate <state-root>
will traverse the whole state from the given root and will abort if any referenced
trie node or contract code is missing. This command can be used for state integrity
verification. The default checking target is the HEAD state. It's basically identical
to traverse-state, but the check granularity is smaller. 

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "dump",
				Usage:     "Dump a specific block from storage (same as 'geth dump' but using snapshots)",
				ArgsUsage: "[? <blockHash> | <blockNum>]",
				Action:    dumpState,
				Flags: qcommon.Merge([]cli.Flag{
					utils.ExcludeCodeFlag,
					utils.ExcludeStorageFlag,
					utils.StartKeyFlag,
					utils.DumpLimitFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
This command is semantically equivalent to 'geth dump', but uses the snapshots
as the backend data source, making this command a lot faster. 

The argument is interpreted as block number or hash. If none is provided, the latest
block is used.
`,
			},
			{
				Name:      "stats",
				Usage:     "Show all snapshot info",
				ArgsUsage: "<root>",
				Action:    stats,
				Flags:     qcommon.Merge(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot stats <state-root>
will traverse the whole shnapshots 
`,
			},
		},
	}
)

func pruneState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)

	defer chaindb.Close()

	if rawdb.ReadStateScheme(chaindb) != rawdb.HashScheme {
		log.Crit("Offline pruning is not required for path scheme")
	}
	prunerconfig := pruner.Config{
		Datadir:   stack.ResolvePath(""),
		BloomSize: ctx.Uint64(utils.BloomFilterSizeFlag.Name),
	}
	pruner, err := pruner.NewPruner(chaindb, prunerconfig)
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return err
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var targetRoot common.Hash
	if ctx.NArg() == 1 {
		targetRoot, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	if err = pruner.Prune(targetRoot); err != nil {
		log.Error("Failed to prune state", "err", err)
		return err
	}
	return nil
}

func verifyState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()
	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	triedb := utils.MakeTrieDatabase(ctx, chaindb, false, true, false)
	defer triedb.Close()

	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, chaindb, triedb, headBlock.Root())
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return err
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var root = headBlock.Root()
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	if err := snaptree.Verify(root); err != nil {
		log.Error("Failed to verify state", "root", root, "err", err)
		return err
	}
	log.Info("Verified the state", "root", root)
	return snapshot.CheckDanglingStorage(chaindb)
}

// checkDanglingStorage iterates the snap storage data, and verifies that all
// storage also has corresponding account data.
func checkDanglingStorage(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	return snapshot.CheckDanglingStorage(utils.MakeChainDatabase(ctx, stack, true))
}

// traverseState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present.
func traverseState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, chaindb, false, true, false)
	defer triedb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}
	t, err := trie.NewStateTrie(trie.StateTrieID(root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", root, "err", err)
		return err
	}
	var (
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
	)
	acctIt, err := t.NodeIterator(nil)
	if err != nil {
		log.Error("Failed to open iterator", "root", root, "err", err)
		return err
	}
	accIter := trie.NewIterator(acctIt)
	for accIter.Next() {
		accounts += 1
		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			log.Error("Invalid account encountered during traversal", "err", err)
			return err
		}
		if acc.Root != types.EmptyRootHash {
			id := trie.StorageTrieID(root, common.BytesToHash(accIter.Key), acc.Root)
			storageTrie, err := trie.NewStateTrie(id, triedb)
			if err != nil {
				log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
				return err
			}
			storageIt, err := storageTrie.NodeIterator(nil)
			if err != nil {
				log.Error("Failed to open storage iterator", "root", acc.Root, "err", err)
				return err
			}
			storageIter := trie.NewIterator(storageIt)
			for storageIter.Next() {
				slots += 1
			}
			if storageIter.Err != nil {
				log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Err)
				return storageIter.Err
			}
		}
		if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
			if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
				log.Error("Code is missing", "hash", common.BytesToHash(acc.CodeHash))
				return errors.New("missing code")
			}
			codes += 1
		}
		if time.Since(lastReport) > time.Second*8 {
			log.Info("Traversing state", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	if accIter.Err != nil {
		log.Error("Failed to traverse state trie", "root", root, "err", accIter.Err)
		return accIter.Err
	}
	log.Info("State is complete", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// traverseRawState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present. It's basically identical to traverseState
// but it will check each trie node.
func traverseRawState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, chaindb, false, true, false)
	defer triedb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}
	t, err := trie.NewStateTrie(trie.StateTrieID(root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", root, "err", err)
		return err
	}
	var (
		nodes      int
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
		hasher     = crypto.NewKeccakState()
		got        = make([]byte, 32)
	)
	accIter, err := t.NodeIterator(nil)
	if err != nil {
		log.Error("Failed to open iterator", "root", root, "err", err)
		return err
	}
	reader, err := triedb.Reader(root)
	if err != nil {
		log.Error("State is non-existent", "root", root)
		return nil
	}
	for accIter.Next(true) {
		nodes += 1
		node := accIter.Hash()

		// Check the present for non-empty hash node(embedded node doesn't
		// have their own hash).
		if node != (common.Hash{}) {
			blob, _ := reader.Node(common.Hash{}, accIter.Path(), node)
			if len(blob) == 0 {
				log.Error("Missing trie node(account)", "hash", node)
				return errors.New("missing account")
			}
			hasher.Reset()
			hasher.Write(blob)
			hasher.Read(got)
			if !bytes.Equal(got, node.Bytes()) {
				log.Error("Invalid trie node(account)", "hash", node.Hex(), "value", blob)
				return errors.New("invalid account node")
			}
		}
		// If it's a leaf node, yes we are touching an account,
		// dig into the storage trie further.
		if accIter.Leaf() {
			accounts += 1
			var acc types.StateAccount
			if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
				log.Error("Invalid account encountered during traversal", "err", err)
				return errors.New("invalid account")
			}
			if acc.Root != types.EmptyRootHash {
				id := trie.StorageTrieID(root, common.BytesToHash(accIter.LeafKey()), acc.Root)
				storageTrie, err := trie.NewStateTrie(id, triedb)
				if err != nil {
					log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
					return errors.New("missing storage trie")
				}
				storageIter, err := storageTrie.NodeIterator(nil)
				if err != nil {
					log.Error("Failed to open storage iterator", "root", acc.Root, "err", err)
					return err
				}
				for storageIter.Next(true) {
					nodes += 1
					node := storageIter.Hash()

					// Check the presence for non-empty hash node(embedded node doesn't
					// have their own hash).
					if node != (common.Hash{}) {
						blob, _ := reader.Node(common.BytesToHash(accIter.LeafKey()), storageIter.Path(), node)
						if len(blob) == 0 {
							log.Error("Missing trie node(storage)", "hash", node)
							return errors.New("missing storage")
						}
						hasher.Reset()
						hasher.Write(blob)
						hasher.Read(got)
						if !bytes.Equal(got, node.Bytes()) {
							log.Error("Invalid trie node(storage)", "hash", node.Hex(), "value", blob)
							return errors.New("invalid storage node")
						}
					}
					// Bump the counter if it's leaf node.
					if storageIter.Leaf() {
						slots += 1
					}
				}
				if storageIter.Error() != nil {
					log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Error())
					return storageIter.Error()
				}
			}
			if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
				if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
					log.Error("Code is missing", "account", common.BytesToHash(accIter.LeafKey()))
					return errors.New("missing code")
				}
				codes += 1
			}
			if time.Since(lastReport) > time.Second*8 {
				log.Info("Traversing state", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
				lastReport = time.Now()
			}
		}
	}
	if accIter.Error() != nil {
		log.Error("Failed to traverse state trie", "root", root, "err", accIter.Error())
		return accIter.Error()
	}
	log.Info("State is complete", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

func parseRoot(input string) (common.Hash, error) {
	var h common.Hash
	if err := h.UnmarshalText([]byte(input)); err != nil {
		return h, err
	}
	return h, nil
}

func dumpState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	conf, db, root, err := parseDumpConfig(ctx, stack)
	if err != nil {
		return err
	}
	triedb := utils.MakeTrieDatabase(ctx, db, false, true, false)
	defer triedb.Close()

	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, db, triedb, root)
	if err != nil {
		return err
	}
	accIt, err := snaptree.AccountIterator(root, common.BytesToHash(conf.Start))
	if err != nil {
		return err
	}
	defer accIt.Release()

	log.Info("Snapshot dumping started", "root", root)
	var (
		start    = time.Now()
		logged   = time.Now()
		accounts uint64
	)
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(struct {
		Root common.Hash `json:"root"`
	}{root})
	for accIt.Next() {
		account, err := types.FullAccount(accIt.Account())
		if err != nil {
			return err
		}
		da := &state.DumpAccount{
			Balance:     account.Balance.String(),
			Nonce:       account.Nonce,
			Root:        account.Root.Bytes(),
			CodeHash:    account.CodeHash,
			AddressHash: accIt.Hash().Bytes(),
		}
		if !conf.SkipCode && !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
			da.Code = rawdb.ReadCode(db, common.BytesToHash(account.CodeHash))
		}
		if !conf.SkipStorage {
			da.Storage = make(map[common.Hash]string)

			stIt, err := snaptree.StorageIterator(root, accIt.Hash(), common.Hash{})
			if err != nil {
				return err
			}
			for stIt.Next() {
				da.Storage[stIt.Hash()] = common.Bytes2Hex(stIt.Slot())
			}
		}
		enc.Encode(da)
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Snapshot dumping in progress", "at", accIt.Hash(), "accounts", accounts,
				"elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
		if conf.Max > 0 && accounts >= conf.Max {
			break
		}
	}
	log.Info("Snapshot dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// checkAccount iterates the snap data layers, and looks up the given account
// across all layers.
func checkAccount(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("need <address|hash> arg")
	}
	var (
		hash common.Hash
		addr common.Address
	)
	switch arg := ctx.Args().First(); len(arg) {
	case 40, 42:
		addr = common.HexToAddress(arg)
		hash = crypto.Keccak256Hash(addr.Bytes())
	case 64, 66:
		hash = common.HexToHash(arg)
	default:
		return errors.New("malformed address or hash")
	}
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()
	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()
	start := time.Now()
	log.Info("Checking difflayer journal", "address", addr, "hash", hash)
	if err := snapshot.CheckJournalAccount(chaindb, hash); err != nil {
		return err
	}
	log.Info("Checked the snapshot journalled storage", "time", common.PrettyDuration(time.Since(start)))
	return nil
}

func parseDumpConfig(ctx *cli.Context, stack *node.Node) (*state.DumpConfig, ethdb.Database, common.Hash, error) {
	db := utils.MakeChainDatabase(ctx, stack, true)
	var header *types.Header
	if ctx.NArg() > 1 {
		return nil, nil, common.Hash{}, fmt.Errorf("expected 1 argument (number or hash), got %d", ctx.NArg())
	}
	if ctx.NArg() == 1 {
		arg := ctx.Args().First()
		if hashish(arg) {
			hash := common.HexToHash(arg)
			if number := rawdb.ReadHeaderNumber(db, hash); number != nil {
				header = rawdb.ReadHeader(db, hash, *number)
			} else {
				return nil, nil, common.Hash{}, fmt.Errorf("block %x not found", hash)
			}
		} else {
			number, err := strconv.ParseUint(arg, 10, 64)
			if err != nil {
				return nil, nil, common.Hash{}, err
			}
			if hash := rawdb.ReadCanonicalHash(db, number); hash != (common.Hash{}) {
				header = rawdb.ReadHeader(db, hash, number)
			} else {
				return nil, nil, common.Hash{}, fmt.Errorf("header for block %d not found", number)
			}
		}
	} else {
		// Use latest
		header = rawdb.ReadHeadHeader(db)
	}
	if header == nil {
		return nil, nil, common.Hash{}, errors.New("no head block found")
	}
	startArg := common.FromHex(ctx.String(utils.StartKeyFlag.Name))
	var start common.Hash
	switch len(startArg) {
	case 0: // common.Hash
	case 32:
		start = common.BytesToHash(startArg)
	case 20:
		start = crypto.Keccak256Hash(startArg)
		log.Info("Converting start-address to hash", "address", common.BytesToAddress(startArg), "hash", start.Hex())
	default:
		return nil, nil, common.Hash{}, fmt.Errorf("invalid start argument: %x. 20 or 32 hex-encoded bytes required", startArg)
	}
	var conf = &state.DumpConfig{
		SkipCode:          ctx.Bool(utils.ExcludeCodeFlag.Name),
		SkipStorage:       ctx.Bool(utils.ExcludeStorageFlag.Name),
		OnlyWithAddresses: !ctx.Bool(utils.IncludeIncompletesFlag.Name),
		Start:             start.Bytes(),
		Max:               ctx.Uint64(utils.DumpLimitFlag.Name),
	}
	log.Info("State dump configured", "block", header.Number, "hash", header.Hash().Hex(),
		"skipcode", conf.SkipCode, "skipstorage", conf.SkipStorage,
		"start", hexutil.Encode(conf.Start), "limit", conf.Max)
	return conf, db, header.Root, nil
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func stats(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx, config.Cfg)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()
	header := rawdb.ReadHeadHeader(chaindb)
	if header == nil {
		log.Error("Failed to load head")
		return errors.New("no head")
	}
	triedb := utils.MakeTrieDatabase(ctx, chaindb, false, true, false)
	defer triedb.Close()

	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, chaindb, triedb, header.Root)
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return err
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var root = header.Root
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	snaps := snaptree.Snapshots(root, math.MaxInt, false)
	log.Info("Snapshots info", "blockRoot", header.Root, "total", len(snaps))
	head := header
	getHead := func(snap snapshot.Snapshot) *types.Header {
		timeout := time.After(time.Minute)
		for head != nil {
			select {
			case <-timeout:
				return nil
			default:
			}
			if snap.Root() == head.Root {
				return head
			}
			if head.Number.Uint64() <= 0 {
				return nil
			}
			head = rawdb.ReadHeader(chaindb, head.ParentHash, head.Number.Uint64()-1)
		}
		return nil
	}
	for i, snap := range snaps {
		shead := getHead(snap)
		if shead != nil {
			log.Info("snapshot", "index", i, "root", snap.Root().String(), "blockHash", shead.Hash().String(), "number", shead.Number.String())
		} else {
			log.Info("snapshot", "index", i, "root", snap.Root().String())
		}
	}
	return nil
}
