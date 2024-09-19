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
	_ = abi.ConvertType
)

// MeerchangeMetaData contains all meta data concerning the Meerchange contract.
var MeerchangeMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"fee\",\"type\":\"uint64\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"sig\",\"type\":\"string\"}],\"name\":\"Export\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Import\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"TO_UTXO_PRECISION\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"txid\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"idx\",\"type\":\"uint32\"},{\"internalType\":\"uint64\",\"name\":\"fee\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"sig\",\"type\":\"string\"}],\"name\":\"export\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getExportCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getImportCount\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getImportTotal\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"importToUtxo\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b50610491806100206000396000f3fe6080604052600436106100655760003560e01c8063994b58f811610043578063994b58f8146100ee578063a8770e6914610107578063da2320261461010f57600080fd5b806347b2c7d71461006a5780635090ea051461008c57806387e9ff23146100cc575b600080fd5b34801561007657600080fd5b50475b6040519081526020015b60405180910390f35b34801561009857600080fd5b5060005468010000000000000000900467ffffffffffffffff165b60405167ffffffffffffffff9091168152602001610083565b3480156100d857600080fd5b506100ec6100e73660046102b1565b61012e565b005b3480156100fa57600080fd5b506100796402540be40081565b6100ec6101b4565b34801561011b57600080fd5b5060005467ffffffffffffffff166100b3565b6000805467ffffffffffffffff16908061014783610360565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507f8de70f4a2cc28f971578497609b5a7bcd97e40800d857fe09d6057bd382c658285858585856040516101a59594939291906103ae565b60405180910390a15050505050565b60006101c56402540be40034610420565b905060008111610235576040517f08c379a000000000000000000000000000000000000000000000000000000000815260206004820181905260248201527f546f205554584f20616d6f756e74206d757374206e6f7420626520656d707479604482015260640160405180910390fd5b6000805468010000000000000000900467ffffffffffffffff1690600861025b83610360565b91906101000a81548167ffffffffffffffff021916908367ffffffffffffffff160217905550507fb9ba2e23b17fbc3f0029c3a6600ef2dd4484bea87a99c7aab54caf84dedcf96b60405160405180910390a150565b6000806000806000608086880312156102c957600080fd5b85359450602086013563ffffffff811681146102e457600080fd5b9350604086013567ffffffffffffffff808216821461030257600080fd5b9093506060870135908082111561031857600080fd5b818801915088601f83011261032c57600080fd5b81358181111561033b57600080fd5b89602082850101111561034d57600080fd5b9699959850939650602001949392505050565b600067ffffffffffffffff8083168181036103a4577f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6001019392505050565b85815263ffffffff8516602082015267ffffffffffffffff8416604082015260806060820152816080820152818360a0830137600081830160a090810191909152601f9092017fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0160101949350505050565b600082610456577f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b50049056fea2646970667358221220d95f125db711bf76c22ef38e58c54fb5c15f6ebe6cd0bdd3062631a03ae6e95464736f6c634300080f0033",
}

// MeerchangeABI is the input ABI used to generate the binding from.
// Deprecated: Use MeerchangeMetaData.ABI instead.
var MeerchangeABI = MeerchangeMetaData.ABI

// MeerchangeBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use MeerchangeMetaData.Bin instead.
var MeerchangeBin = MeerchangeMetaData.Bin

// DeployMeerchange deploys a new Ethereum contract, binding an instance of Meerchange to it.
func DeployMeerchange(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Meerchange, error) {
	parsed, err := MeerchangeMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(MeerchangeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Meerchange{MeerchangeCaller: MeerchangeCaller{contract: contract}, MeerchangeTransactor: MeerchangeTransactor{contract: contract}, MeerchangeFilterer: MeerchangeFilterer{contract: contract}}, nil
}

// Meerchange is an auto generated Go binding around an Ethereum contract.
type Meerchange struct {
	MeerchangeCaller     // Read-only binding to the contract
	MeerchangeTransactor // Write-only binding to the contract
	MeerchangeFilterer   // Log filterer for contract events
}

// MeerchangeCaller is an auto generated read-only Go binding around an Ethereum contract.
type MeerchangeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MeerchangeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MeerchangeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MeerchangeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MeerchangeSession struct {
	Contract     *Meerchange       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MeerchangeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MeerchangeCallerSession struct {
	Contract *MeerchangeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// MeerchangeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MeerchangeTransactorSession struct {
	Contract     *MeerchangeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// MeerchangeRaw is an auto generated low-level Go binding around an Ethereum contract.
type MeerchangeRaw struct {
	Contract *Meerchange // Generic contract binding to access the raw methods on
}

// MeerchangeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MeerchangeCallerRaw struct {
	Contract *MeerchangeCaller // Generic read-only contract binding to access the raw methods on
}

// MeerchangeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MeerchangeTransactorRaw struct {
	Contract *MeerchangeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMeerchange creates a new instance of Meerchange, bound to a specific deployed contract.
func NewMeerchange(address common.Address, backend bind.ContractBackend) (*Meerchange, error) {
	contract, err := bindMeerchange(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Meerchange{MeerchangeCaller: MeerchangeCaller{contract: contract}, MeerchangeTransactor: MeerchangeTransactor{contract: contract}, MeerchangeFilterer: MeerchangeFilterer{contract: contract}}, nil
}

// NewMeerchangeCaller creates a new read-only instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeCaller(address common.Address, caller bind.ContractCaller) (*MeerchangeCaller, error) {
	contract, err := bindMeerchange(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MeerchangeCaller{contract: contract}, nil
}

// NewMeerchangeTransactor creates a new write-only instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeTransactor(address common.Address, transactor bind.ContractTransactor) (*MeerchangeTransactor, error) {
	contract, err := bindMeerchange(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MeerchangeTransactor{contract: contract}, nil
}

// NewMeerchangeFilterer creates a new log filterer instance of Meerchange, bound to a specific deployed contract.
func NewMeerchangeFilterer(address common.Address, filterer bind.ContractFilterer) (*MeerchangeFilterer, error) {
	contract, err := bindMeerchange(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MeerchangeFilterer{contract: contract}, nil
}

// bindMeerchange binds a generic wrapper to an already deployed contract.
func bindMeerchange(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := MeerchangeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Meerchange *MeerchangeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Meerchange.Contract.MeerchangeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Meerchange *MeerchangeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Meerchange.Contract.MeerchangeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Meerchange *MeerchangeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Meerchange.Contract.MeerchangeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Meerchange *MeerchangeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Meerchange.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Meerchange *MeerchangeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Meerchange.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Meerchange *MeerchangeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Meerchange.Contract.contract.Transact(opts, method, params...)
}

// TOUTXOPRECISION is a free data retrieval call binding the contract method 0x994b58f8.
//
// Solidity: function TO_UTXO_PRECISION() view returns(uint256)
func (_Meerchange *MeerchangeCaller) TOUTXOPRECISION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "TO_UTXO_PRECISION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TOUTXOPRECISION is a free data retrieval call binding the contract method 0x994b58f8.
//
// Solidity: function TO_UTXO_PRECISION() view returns(uint256)
func (_Meerchange *MeerchangeSession) TOUTXOPRECISION() (*big.Int, error) {
	return _Meerchange.Contract.TOUTXOPRECISION(&_Meerchange.CallOpts)
}

// TOUTXOPRECISION is a free data retrieval call binding the contract method 0x994b58f8.
//
// Solidity: function TO_UTXO_PRECISION() view returns(uint256)
func (_Meerchange *MeerchangeCallerSession) TOUTXOPRECISION() (*big.Int, error) {
	return _Meerchange.Contract.TOUTXOPRECISION(&_Meerchange.CallOpts)
}

// GetExportCount is a free data retrieval call binding the contract method 0xda232026.
//
// Solidity: function getExportCount() view returns(uint64)
func (_Meerchange *MeerchangeCaller) GetExportCount(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "getExportCount")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetExportCount is a free data retrieval call binding the contract method 0xda232026.
//
// Solidity: function getExportCount() view returns(uint64)
func (_Meerchange *MeerchangeSession) GetExportCount() (uint64, error) {
	return _Meerchange.Contract.GetExportCount(&_Meerchange.CallOpts)
}

// GetExportCount is a free data retrieval call binding the contract method 0xda232026.
//
// Solidity: function getExportCount() view returns(uint64)
func (_Meerchange *MeerchangeCallerSession) GetExportCount() (uint64, error) {
	return _Meerchange.Contract.GetExportCount(&_Meerchange.CallOpts)
}

// GetImportCount is a free data retrieval call binding the contract method 0x5090ea05.
//
// Solidity: function getImportCount() view returns(uint64)
func (_Meerchange *MeerchangeCaller) GetImportCount(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "getImportCount")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetImportCount is a free data retrieval call binding the contract method 0x5090ea05.
//
// Solidity: function getImportCount() view returns(uint64)
func (_Meerchange *MeerchangeSession) GetImportCount() (uint64, error) {
	return _Meerchange.Contract.GetImportCount(&_Meerchange.CallOpts)
}

// GetImportCount is a free data retrieval call binding the contract method 0x5090ea05.
//
// Solidity: function getImportCount() view returns(uint64)
func (_Meerchange *MeerchangeCallerSession) GetImportCount() (uint64, error) {
	return _Meerchange.Contract.GetImportCount(&_Meerchange.CallOpts)
}

// GetImportTotal is a free data retrieval call binding the contract method 0x47b2c7d7.
//
// Solidity: function getImportTotal() view returns(uint256)
func (_Meerchange *MeerchangeCaller) GetImportTotal(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Meerchange.contract.Call(opts, &out, "getImportTotal")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetImportTotal is a free data retrieval call binding the contract method 0x47b2c7d7.
//
// Solidity: function getImportTotal() view returns(uint256)
func (_Meerchange *MeerchangeSession) GetImportTotal() (*big.Int, error) {
	return _Meerchange.Contract.GetImportTotal(&_Meerchange.CallOpts)
}

// GetImportTotal is a free data retrieval call binding the contract method 0x47b2c7d7.
//
// Solidity: function getImportTotal() view returns(uint256)
func (_Meerchange *MeerchangeCallerSession) GetImportTotal() (*big.Int, error) {
	return _Meerchange.Contract.GetImportTotal(&_Meerchange.CallOpts)
}

// Export is a paid mutator transaction binding the contract method 0x87e9ff23.
//
// Solidity: function export(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeTransactor) Export(opts *bind.TransactOpts, txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.contract.Transact(opts, "export", txid, idx, fee, sig)
}

// Export is a paid mutator transaction binding the contract method 0x87e9ff23.
//
// Solidity: function export(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeSession) Export(txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.Contract.Export(&_Meerchange.TransactOpts, txid, idx, fee, sig)
}

// Export is a paid mutator transaction binding the contract method 0x87e9ff23.
//
// Solidity: function export(bytes32 txid, uint32 idx, uint64 fee, string sig) returns()
func (_Meerchange *MeerchangeTransactorSession) Export(txid [32]byte, idx uint32, fee uint64, sig string) (*types.Transaction, error) {
	return _Meerchange.Contract.Export(&_Meerchange.TransactOpts, txid, idx, fee, sig)
}

// ImportToUtxo is a paid mutator transaction binding the contract method 0xa8770e69.
//
// Solidity: function importToUtxo() payable returns()
func (_Meerchange *MeerchangeTransactor) ImportToUtxo(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Meerchange.contract.Transact(opts, "importToUtxo")
}

// ImportToUtxo is a paid mutator transaction binding the contract method 0xa8770e69.
//
// Solidity: function importToUtxo() payable returns()
func (_Meerchange *MeerchangeSession) ImportToUtxo() (*types.Transaction, error) {
	return _Meerchange.Contract.ImportToUtxo(&_Meerchange.TransactOpts)
}

// ImportToUtxo is a paid mutator transaction binding the contract method 0xa8770e69.
//
// Solidity: function importToUtxo() payable returns()
func (_Meerchange *MeerchangeTransactorSession) ImportToUtxo() (*types.Transaction, error) {
	return _Meerchange.Contract.ImportToUtxo(&_Meerchange.TransactOpts)
}

// MeerchangeExportIterator is returned from FilterExport and is used to iterate over the raw logs and unpacked data for Export events raised by the Meerchange contract.
type MeerchangeExportIterator struct {
	Event *MeerchangeExport // Event containing the contract specifics and raw log

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
func (it *MeerchangeExportIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MeerchangeExport)
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
		it.Event = new(MeerchangeExport)
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
func (it *MeerchangeExportIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MeerchangeExportIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MeerchangeExport represents a Export event raised by the Meerchange contract.
type MeerchangeExport struct {
	Txid [32]byte
	Idx  uint32
	Fee  uint64
	Sig  string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterExport is a free log retrieval operation binding the contract event 0x8de70f4a2cc28f971578497609b5a7bcd97e40800d857fe09d6057bd382c6582.
//
// Solidity: event Export(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) FilterExport(opts *bind.FilterOpts) (*MeerchangeExportIterator, error) {

	logs, sub, err := _Meerchange.contract.FilterLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return &MeerchangeExportIterator{contract: _Meerchange.contract, event: "Export", logs: logs, sub: sub}, nil
}

// WatchExport is a free log subscription operation binding the contract event 0x8de70f4a2cc28f971578497609b5a7bcd97e40800d857fe09d6057bd382c6582.
//
// Solidity: event Export(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) WatchExport(opts *bind.WatchOpts, sink chan<- *MeerchangeExport) (event.Subscription, error) {

	logs, sub, err := _Meerchange.contract.WatchLogs(opts, "Export")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MeerchangeExport)
				if err := _Meerchange.contract.UnpackLog(event, "Export", log); err != nil {
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

// ParseExport is a log parse operation binding the contract event 0x8de70f4a2cc28f971578497609b5a7bcd97e40800d857fe09d6057bd382c6582.
//
// Solidity: event Export(bytes32 txid, uint32 idx, uint64 fee, string sig)
func (_Meerchange *MeerchangeFilterer) ParseExport(log types.Log) (*MeerchangeExport, error) {
	event := new(MeerchangeExport)
	if err := _Meerchange.contract.UnpackLog(event, "Export", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// MeerchangeImportIterator is returned from FilterImport and is used to iterate over the raw logs and unpacked data for Import events raised by the Meerchange contract.
type MeerchangeImportIterator struct {
	Event *MeerchangeImport // Event containing the contract specifics and raw log

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
func (it *MeerchangeImportIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MeerchangeImport)
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
		it.Event = new(MeerchangeImport)
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
func (it *MeerchangeImportIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MeerchangeImportIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MeerchangeImport represents a Import event raised by the Meerchange contract.
type MeerchangeImport struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterImport is a free log retrieval operation binding the contract event 0xb9ba2e23b17fbc3f0029c3a6600ef2dd4484bea87a99c7aab54caf84dedcf96b.
//
// Solidity: event Import()
func (_Meerchange *MeerchangeFilterer) FilterImport(opts *bind.FilterOpts) (*MeerchangeImportIterator, error) {

	logs, sub, err := _Meerchange.contract.FilterLogs(opts, "Import")
	if err != nil {
		return nil, err
	}
	return &MeerchangeImportIterator{contract: _Meerchange.contract, event: "Import", logs: logs, sub: sub}, nil
}

// WatchImport is a free log subscription operation binding the contract event 0xb9ba2e23b17fbc3f0029c3a6600ef2dd4484bea87a99c7aab54caf84dedcf96b.
//
// Solidity: event Import()
func (_Meerchange *MeerchangeFilterer) WatchImport(opts *bind.WatchOpts, sink chan<- *MeerchangeImport) (event.Subscription, error) {

	logs, sub, err := _Meerchange.contract.WatchLogs(opts, "Import")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MeerchangeImport)
				if err := _Meerchange.contract.UnpackLog(event, "Import", log); err != nil {
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

// ParseImport is a log parse operation binding the contract event 0xb9ba2e23b17fbc3f0029c3a6600ef2dd4484bea87a99c7aab54caf84dedcf96b.
//
// Solidity: event Import()
func (_Meerchange *MeerchangeFilterer) ParseImport(log types.Log) (*MeerchangeImport, error) {
	event := new(MeerchangeImport)
	if err := _Meerchange.contract.UnpackLog(event, "Import", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
