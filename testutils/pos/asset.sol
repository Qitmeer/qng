// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Asset {

    address public owner;

    struct MinerInfo {
        uint256 startTime;
        bool enable;
        uint256 rewardPerSec;
        uint256 lastMiningTime;
    }

    // var
    mapping(address => MinerInfo) miners; // miners

    // events
    event Mining(address indexed _miner, uint256 _value);
    event SetMiner(address indexed _miner, MinerInfo mi);
    event Deposit(address indexed _user, uint256 _value);

    constructor() {
        owner = msg.sender;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, 'not owner!!!');
        _;
    }

    modifier onlyMiner() {
        require(miners[msg.sender].enable, 'not miner!!!');
        _;
    }

    // contract can receive meer
    receive() external payable {
        emit Deposit(msg.sender, msg.value);
    }

    function setMiner(uint256 startTime,bool start,address miner,uint256 _reward) external onlyOwner {
        miners[miner].enable = start;
        miners[miner].startTime = startTime;
        miners[miner].rewardPerSec = _reward;
        miners[miner].lastMiningTime = block.timestamp;
        emit SetMiner(miner,miners[miner]);
    }

    function mining() external onlyMiner payable{
        require(miners[msg.sender].enable,"not start");
        require(block.timestamp >= miners[msg.sender].startTime,"not start");
        if(block.timestamp <= miners[msg.sender].lastMiningTime){
            return;
        }
        uint256 canMining = (block.timestamp - miners[msg.sender].lastMiningTime) * miners[msg.sender].rewardPerSec;
        miners[msg.sender].lastMiningTime = block.timestamp;
        if (canMining > payable(address(this)).balance){
            canMining = payable(address(this)).balance;
        }
        if(canMining <= 0){
            return;
        }
        payable(msg.sender).transfer(canMining);
        emit Mining(msg.sender,canMining);
    }
}