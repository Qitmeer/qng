// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package meerchange

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// CrosschainMetaData contains all meta data concerning the Crosschain contract.
var CrosschainMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"Export\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"}],\"name\":\"export\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getExport\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getImport\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610332806100206000396000f3fe608060405234801561001057600080fd5b50600436106100415760003560e01c80634ccccead14610046578063ac55902914610062578063bf2ec2e614610080575b600080fd5b610060600480360381019061005b91906101d8565b61009e565b005b61006a610127565b604051610077919061023b565b60405180910390f35b610088610144565b604051610095919061023b565b60405180910390f35b60008081819054906101000a900467ffffffffffffffff16809291906100c390610285565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507fa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39828260405161011b9291906102d3565b60405180910390a15050565b60008060089054906101000a900467ffffffffffffffff16905090565b60008060009054906101000a900467ffffffffffffffff16905090565b600080fd5b6000819050919050565b61017981610166565b811461018457600080fd5b50565b60008135905061019681610170565b92915050565b600063ffffffff82169050919050565b6101b58161019c565b81146101c057600080fd5b50565b6000813590506101d2816101ac565b92915050565b600080604083850312156101ef576101ee610161565b5b60006101fd85828601610187565b925050602061020e858286016101c3565b9150509250929050565b600067ffffffffffffffff82169050919050565b61023581610218565b82525050565b6000602082019050610250600083018461022c565b92915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061029082610218565b915067ffffffffffffffff82036102aa576102a9610256565b5b600182019050919050565b6102be81610166565b82525050565b6102cd8161019c565b82525050565b60006040820190506102e860008301856102b5565b6102f560208301846102c4565b939250505056fea26469706673582212209b77f2568c28ff8de1cb99795c675ce252e720fd1abb541cd6130f7af618ab5a64736f6c634300080d0033",
}

// CrosschainABI is the input ABI used to generate the binding from.
// Deprecated: Use CrosschainMetaData.ABI instead.
var CrosschainABI = CrosschainMetaData.ABI

// CrosschainBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CrosschainMetaData.Bin instead.
var CrosschainBin = CrosschainMetaData.Bin

// DeployCrosschain deploys a new Ethereum contract, binding an instance of Crosschain to it.
func DeployCrosschain(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Crosschain, error) {
	parsed, err := CrosschainMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CrosschainBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Crosschain{CrosschainCaller: CrosschainCaller{contract: contract}, CrosschainTransactor: CrosschainTransactor{contract: contract}, CrosschainFilterer: CrosschainFilterer{contract: contract}}, nil
}

// Crosschain is an auto generated Go binding around an Ethereum contract.
type Crosschain struct {
	CrosschainCaller     // Read-only binding to the contract
	CrosschainTransactor // Write-only binding to the contract
	CrosschainFilterer   // Log filterer for contract events
}

// CrosschainCaller is an auto generated read-only Go binding around an Ethereum contract.
type CrosschainCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrosschainTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CrosschainTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrosschainFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CrosschainFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CrosschainSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CrosschainSession struct {
	Contract     *Crosschain       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CrosschainCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CrosschainCallerSession struct {
	Contract *CrosschainCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// CrosschainTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CrosschainTransactorSession struct {
	Contract     *CrosschainTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// CrosschainRaw is an auto generated low-level Go binding around an Ethereum contract.
type CrosschainRaw struct {
	Contract *Crosschain // Generic contract binding to access the raw methods on
}

// CrosschainCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CrosschainCallerRaw struct {
	Contract *CrosschainCaller // Generic read-only contract binding to access the raw methods on
}

// CrosschainTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CrosschainTransactorRaw struct {
	Contract *CrosschainTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCrosschain creates a new instance of Crosschain, bound to a specific deployed contract.
func NewCrosschain(address common.Address, backend bind.ContractBackend) (*Crosschain, error) {
	contract, err := bindCrosschain(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Crosschain{CrosschainCaller: CrosschainCaller{contract: contract}, CrosschainTransactor: CrosschainTransactor{contract: contract}, CrosschainFilterer: CrosschainFilterer{contract: contract}}, nil
}

// NewCrosschainCaller creates a new read-only instance of Crosschain, bound to a specific deployed contract.
func NewCrosschainCaller(address common.Address, caller bind.ContractCaller) (*CrosschainCaller, error) {
	contract, err := bindCrosschain(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CrosschainCaller{contract: contract}, nil
}

// NewCrosschainTransactor creates a new write-only instance of Crosschain, bound to a specific deployed contract.
func NewCrosschainTransactor(address common.Address, transactor bind.ContractTransactor) (*CrosschainTransactor, error) {
	contract, err := bindCrosschain(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CrosschainTransactor{contract: contract}, nil
}

// NewCrosschainFilterer creates a new log filterer instance of Crosschain, bound to a specific deployed contract.
func NewCrosschainFilterer(address common.Address, filterer bind.ContractFilterer) (*CrosschainFilterer, error) {
	contract, err := bindCrosschain(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CrosschainFilterer{contract: contract}, nil
}

// bindCrosschain binds a generic wrapper to an already deployed contract.
func bindCrosschain(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(CrosschainABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Crosschain *CrosschainRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Crosschain.Contract.CrosschainCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Crosschain *CrosschainRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Crosschain.Contract.CrosschainTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Crosschain *CrosschainRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Crosschain.Contract.CrosschainTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Crosschain *CrosschainCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Crosschain.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Crosschain *CrosschainTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Crosschain.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Crosschain *CrosschainTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Crosschain.Contract.contract.Transact(opts, method, params...)
}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Crosschain *CrosschainCaller) GetExport(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Crosschain.contract.Call(opts, &out, "getExport")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Crosschain *CrosschainSession) GetExport() (uint64, error) {
	return _Crosschain.Contract.GetExport(&_Crosschain.CallOpts)
}

// GetExport is a free data retrieval call binding the contract method 0xbf2ec2e6.
//
// Solidity: function getExport() view returns(uint64)
func (_Crosschain *CrosschainCallerSession) GetExport() (uint64, error) {
	return _Crosschain.Contract.GetExport(&_Crosschain.CallOpts)
}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Crosschain *CrosschainCaller) GetImport(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Crosschain.contract.Call(opts, &out, "getImport")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Crosschain *CrosschainSession) GetImport() (uint64, error) {
	return _Crosschain.Contract.GetImport(&_Crosschain.CallOpts)
}

// GetImport is a free data retrieval call binding the contract method 0xac559029.
//
// Solidity: function getImport() view returns(uint64)
func (_Crosschain *CrosschainCallerSession) GetImport() (uint64, error) {
	return _Crosschain.Contract.GetImport(&_Crosschain.CallOpts)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Crosschain *CrosschainTransactor) Export(opts *bind.TransactOpts, txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Crosschain.contract.Transact(opts, "export", txid, idx)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Crosschain *CrosschainSession) Export(txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Crosschain.Contract.Export(&_Crosschain.TransactOpts, txid, idx)
}

// Export is a paid mutator transaction binding the contract method 0x4ccccead.
//
// Solidity: function export(bytes32 txid, uint32 idx) returns()
func (_Crosschain *CrosschainTransactorSession) Export(txid [32]byte, idx uint32) (*types.Transaction, error) {
	return _Crosschain.Contract.Export(&_Crosschain.TransactOpts, txid, idx)
}

// CrosschainExportIterator is returned from FilterExport and is used to iterate over the raw logs and unpacked data for Export events raised by the Crosschain contract.
type CrosschainExportIterator struct {
	Event *CrosschainExport // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *CrosschainExportIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CrosschainExport)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(CrosschainExport)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *CrosschainExportIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CrosschainExportIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CrosschainExport represents a Export event raised by the Crosschain contract.
type CrosschainExport struct {
	Txid [32]byte
	Idx  uint32
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterExport is a free log retrieval operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Crosschain *CrosschainFilterer) FilterExport(opts *bind.FilterOpts) (*CrosschainExportIterator, error) {

	logs, sub, err := _Crosschain.contract.FilterLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return &CrosschainExportIterator{contract: _Crosschain.contract, event: "Export", logs: logs, sub: sub}, nil
}

// WatchExport is a free log subscription operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Crosschain *CrosschainFilterer) WatchExport(opts *bind.WatchOpts, sink chan<- *CrosschainExport) (event.Subscription, error) {

	logs, sub, err := _Crosschain.contract.WatchLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CrosschainExport)
				if err := _Crosschain.contract.UnpackLog(event, "Export", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseExport is a log parse operation binding the contract event 0xa752968fe9a336e1e2de83b6e2f3bc1dd7d05dd3359dd92d9b92993209fffe39.
//
// Solidity: event Export(bytes32 txid, uint32 idx)
func (_Crosschain *CrosschainFilterer) ParseExport(log types.Log) (*CrosschainExport, error) {
	event := new(CrosschainExport)
	if err := _Crosschain.contract.UnpackLog(event, "Export", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
