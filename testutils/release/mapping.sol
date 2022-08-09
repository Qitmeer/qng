// SPDX-License-Identifier: MIT
pragma solidity ^0.8.3;

contract MeerMapping{

    // all lockUsers mapping Amounts
    mapping(bytes => uint256) public meerMappingAmounts; // index 0

    // query amount
    function queryAmount(bytes memory _qngHash160) external view returns (uint256) {
        return meerMappingAmounts[_qngHash160];
    }

}