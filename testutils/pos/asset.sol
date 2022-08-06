// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Asset {
    // var
    uint256 public startTime; // mining start time
    bool public start; // start time

    mapping(address => uint256) miners; // miners

    uint256 public allPower; // all power
    uint256 public lastMiningTime; // last mining time
    uint256 public rewardPerSec; // reward per second

    // events
    event Mining(address indexed _miner, uint256 _value);
    event SetMiner(address indexed _miner, uint256 _power);

    constructor() {
        owner = msg.sender;
    }

    modifier onlyOwner() {
        require(msg.sender == owner, 'not owner!!!');
        _;
    }

    modifier onlyMiner() {
        require(miners[msg.sender] > 0, 'not miner!!!');
        _;
    }

    // contract can receive meer
    receive() external payable {
        emit Deposit(msg.sender, msg.value);
    }

    // set release start time
    function setStartTime(uint256 _start) external onlyOwner {
        startTime = _start;
    }

    function setStart(bool _start) external onlyOwner {
        start = _start;
    }

    function setRewardPerSec(bool _reward) external onlyOwner {
        rewardPerSec = _reward;
    }

    function setMiner(address miner ,uint256 _power) external onlyOwner {
        allPower -= miners[miner];
        miners[miner] = _power;
        allPower += _power;
        emit SetMiner(miner,_power);
    }

    function mining() external onlyMiner payable{
        require(start,"not start");
        require(allPower > 0,"not start");
        require(block.timestamp >= startTime,"not start");
        require(block.timestamp >= lastMiningTime,"not start");
        if(block.timestamp <= lastMiningTime){
            return;
        }
        uint256 canMining = (block.timestamp - lastMiningTime) * rewardPerSec;
        lastMiningTime = block.timestamp;
        canMining = canMining * miners[msg.sender] / allPower;
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