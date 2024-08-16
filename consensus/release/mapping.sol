// SPDX-License-Identifier: MIT
pragma solidity ^0.8.3;

contract MeerMapping{

    struct BurnDetail {
        uint256 Amount;
        uint256 Time;
        uint256 Order;
        uint256 Height;
    }
    mapping(bytes => uint256) public meerMappingCount; // index 0

    // all lockUsers mapping Amounts
    mapping(bytes => mapping(uint256 => BurnDetail)) public meerMappingAmounts;

    // query amount
    function queryAmount(bytes memory _qngHash160) external view returns (uint256) {
        uint256 a = 0;
        for (uint256 i=0;i<meerMappingCount[_qngHash160];i++){
            a += meerMappingAmounts[_qngHash160][i].Amount;
        }
        return a;
    }

    // query burn detail
    function queryBurnDetails(bytes memory _qngHash160) external view returns (BurnDetail[] memory) {
        BurnDetail[] memory bd = new BurnDetail[](meerMappingCount[_qngHash160]);
        for (uint256 i=0;i<meerMappingCount[_qngHash160];i++){
            bd[i] = meerMappingAmounts[_qngHash160][i];
        }
        return bd;
    }

}