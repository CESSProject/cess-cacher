# cess-cahcer

## Introduction

Cacher is an important part of CESS CDN, which is used to improve the speed of users downloading files. The cacher is built between the user and the CESS storage miner. The user creates the cache order by indexer, and then downloads the cache file from the cache miner.

## Get start

1. First, you need to make a simple configuration. The configuration file is config.toml under the config directory,Please fill in all configuration options.

```toml
#Directory where the file cache is stored 
CacheDir=""
#There are some data for configuring the cache function
#MaxCacherSize represents the maximum cache space you allow cacher to use(byte)
MaxCacheSize=107374182400
#MaxCacheRate indicates the maximum utilization of cache space. If this threshold is exceeded, files will be cleaned up according to the cache obsolescence policy
MaxCacheRate=0.95
#Threshold indicates the target threshold when cache obsolescence occurs, that is, when cache space utilization reaches this value, cache clean will be stopped
Threshold=0.8
#FreqWeight represents the weight of file usage frequency, which is used in cache obsolescence strategy
FreqWeight=0.3
#cacher IP address,please ensure external accessibility
ServerIp=""
#cacher server port
ServerPort="8080"
#the key used to encrypt the token, which is generated randomly by default
TokenKey=""
#you CESS account and seed
AccountSeed="lunar talent spend shield blade when dumb toilet drastic unique taxi water"
AccountID="cXgZo3RuYkAGhhvCHjAcc9FU13CG44oy8xW6jN39UYvbBaJx5"
#CESS network ws address
RpcAddr="wss://devnet-rpc.cess.cloud/ws/"
#unit price of bytes downloaded from file cache
BytePrice=1000
```

2. Before starting the cache service, you need to register the cache miner,you need to go back to the project main directory and run:

	```shell
	go run main.go register
	```

3. You can run the following command to update the registration information:

	```shell
	go run main.go update
	```

4. And you can run the following command to logout:

	```shell
	go run main.go logout
	```
## Unit Test
You can use the test samples in the test directory for unit testing. Note that you should set the configuration file before testing
```shell
cd test 
# test cacher chain client
go test chain_test.go
# test cacher init
go test init_test.go
# test cacher query api
go test query_test.go
```
## Run Cache Server

You only need to start the cache service with one line of command, and the subsequent tasks should be handed to the indexer. Of course, cache miners also provide a series of rich APIs for developers to use, which will be explained later.

```shell
go run main.go run
```



