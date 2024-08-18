//SPDX-License-Identifier: Unlicense
pragma solidity ^0.8.0;

contract MeerChange {
    // Convert to UTXO precision
    uint256 public constant TO_UTXO_PRECISION = 1e10;
    // The count of call export
    uint64 private exportCount;
    // The count of call import
    uint64 private importCount;

    // events
    event Export(
        bytes32 txid,
        uint32 idx
    ); 

    event Export4337(
        bytes32 txid,
        uint32 idx,
        uint64 fee,
        string sig
    ); 

    event Import(
    ); 

    // Export amount from UTXO
    function export(bytes32 txid,uint32 idx) public {
        exportCount++;
        emit Export(txid,idx);
    }

    // Export amount from UTXO by EIP-4337
    function export4337(bytes32 txid,uint32 idx,uint64 fee,string calldata sig) public {
        exportCount++;
        emit Export4337(txid,idx,fee,sig);
    }

    // Get the count of export
    function getExportCount() public view returns (uint64) {
        return exportCount;
    }

    // Import to UTXO account system
    function importToUtxo() external payable {
        uint256 up = msg.value/TO_UTXO_PRECISION;
        require(up > 0, "To UTXO amount must not be empty");
        importCount++;
        emit Import();
    }

    // Get the count of import
    function getImportCount() public view returns (uint64) {
        return importCount;
    }

    // Get the total of import amount
    function getImportTotal() external view returns (uint256) {
        return address(this).balance;
    }
}
