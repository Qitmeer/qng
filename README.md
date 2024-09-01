# qng
The next generation of the Qitmeer network implementation with the plug-able VMs under the MeerDAG consensus.

### Installation
***make sure you have go-lang version `1.21.x` or above (`1.22.x` is recommended) installed on the system to build.***

* Build from source  
```bash
~ git clone https://github.com/Qitmeer/qng.git
~ cd qng
~ make
```

or
* Install the latest qng available here:
https://github.com/Qitmeer/qng/releases 

or
* Build with Docker:
```bash
~ docker build -t qng .
```

### Make sure you have enough memory  

***Note : Swap 4G+ is required for running the mainent node otherwise you might need the 8G physical memory.***

#### Ex: How to setup a 4G+ swap file on an ubuntu machine

1. setup the swap file
```
sudo apt update
sudo apt install --reinstall util-linux
sudo fallocate -l 4G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

2. check the running swap file
```
sudo swapon -s
```

3. edit `/etc/fstab` file (optional) 
```
sudo vi /etc/fstab
```
add a line to the end of `/etc/fstab` file
```
/swapfile swap swap defaults 0 0
```


### Getting Started
* We take the construction of test network nodes as an example:
```
~ cd ./build/bin
~ ./qng --testnet
~ docker run --rm -it --name qng qng:latest ./build/bin/qng --mixnet --acceptnonstd --modules=qitmeer --modules=p2p
``` 




### Miner

* If you are a miner, you also need to configure your reward address:
```
~ ./qng --testnet --miningaddr=Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx 
``` 

*  Please note that the mining address here is a PK address:
```
~ ./qx ec-to-public [Your_Private_Key] | ./qx ec-to-pkaddr -v=testnet
``` 
*  If you use the old address(`PKH Address`), you will only be unable to package the cross chain transaction.

### Address
##### Use qx Command line tools 
Note: to use "qx" tool, you need to enter the directory ./cmd/qx to build the qx command, see [this guide](https://github.com/Qitmeer/qng/tree/main/cmd/qx#readme) for details
* PKH Address:
```
~ ./qx ec-to-public [Your_Private_Key] | ./qx ec-to-addr -v=testnet
```
* PK Address:
```
~ ./qx ec-to-public [Your_Private_Key] | ./qx ec-to-pkaddr -v=testnet
```
* MeerDAG Address:
```
~ ./qx ec-to-public [Your_Private_Key] | ./qx ec-to-ethaddr
or
~ ./qx pkaddr-to-public [Your_pkaddress] | ./qx ec-to-ethaddr
or
~ ./qx pkaddr-to-ethaddr [Your_pkaddress]
```


##### Use qng RPC 
* All addresses corresponding to the same private key: 
```
~ ./cli.sh getaddresses [Your_Private_Key]
```
(Due to safety reasons, you need to actively open the private module by `./qng --modules=test`)

### MeerEVM
* If you want to use our MeerEVM function, the required interface information can be queried in this RPC:
```
~ cd ./script
~ ./cli.sh vmsinfo
``` 
* If you don't need the default configuration, we provide an environment configuration parameter to meet your custom configuration for MeerEVM:
```
~ ./qng --testnet --evmenv="--http"
or
~ ./qng --testnet --evmenv="--http --http.port=18545 --ws --ws.port=18546"
~ 
``` 


* You first need to transfer your money in qitmeer to MeerEVM:`createExportRawTx <txid> <vout> <PKAdress> <amount>`
``` 
~ ./cli.sh createExportRawTx ce28ec92cc99b13d9f7a658d2f1e08aa9e4f27ebcfaf5344750bb77484a79657 0 Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx 11000000000
~ ./cli.sh txSign [Your_Private_Key] [rawTx]
~ ./cli.sh sendRawTx [signRawTx]
``` 
* Or you can use the multiple inputs or outputs version:`createExportRawTxV2 <inputs> <outputs> <lockTime>`
``` 
~ ./cli.sh createExportRawTxV2 '[{"txid":"0e6aa3a41c6712ed5d68960f2315041579767a9d0a7be9988276cc802e2ae269","vout":0},{"txid":"2d1b3e5e89fbcec54368b7d98079bf533e38f1ce48bfd752582ea87bbac5cbca","vout":0}],[{"address":"Tk6tMafZQW1r2WzwW9V8ynq2HkLhc43nPaMivHTsJGvBUHRNLycPh","amount":11000000000},{"address":"TnNbgxLpoPJCLTcsJbHCzpzcHUouTtfbP8c","amount":999900000}]' 
~ ./cli.sh txSign [Your_Private_Key] [rawTx]
~ ./cli.sh sendRawTx [signRawTx]
``` 
* Finally, wait for the miner to pack your transaction into the block. Then you have the money to start operating your MeerEVM ecosystem.


### How can I transfer my money in meerevm to the qitmeer account system ?
```
~ ./cli.sh createImportRawTx Tk6uXJ3kjh3yA4q94KQF9DTL14rDbd4vb2kztbkfhMBziR35HYkkx [amount] 
~ ./cli.sh txSign [Your_Private_Key] [rawTx]
~ ./cli.sh sendRawTx [signRawTx]
``` 
* Finally, wait for the miner to pack your transaction into the block. 

### How to call QNG's RPC in the JavaScript runtime environment of meerevm ?
```
~ ./qng --testnet --evmenv="--http --http.port=18545 --http.api=net,web3,eth,qng"
~ ./qng attach http://127.0.0.1:18545

Welcome to the Geth JavaScript console!

instance: meereth/v1.10.9-stable/darwin-amd64/go1.16.2
at block: 0 (Thu Jan 01 1970 08:00:00 GMT+0800 (CST))
 datadir: /bin/data/testnet
 modules: eth:1.0 net:1.0 qng:1.0 rpc:1.0 web3:1.0

To exit, press ctrl-d or type exit
> qng.getNodeInfo
...
...

``` 

### How to export the data of blocks from node
```
~ ./qng blockchain export
or
~ ./qng blockchain export --path=[Output directory]
```

### How to import the data of blocks to node
```
~ ./qng blockchain import
or
~ ./qng blockchain import --path=[Input directory]
```

### How to upgrade the data of blocks to node

```
~ ./qng --testnet blockchain upgrade
```

### First aid mode under consensus error

```
~ ./qng --testnet blockchain upgrade --aidmode
```

#### How to configure a custom amana blockchain

```
~ ./qng --amana --amanagen="./custom_amana.json"
```