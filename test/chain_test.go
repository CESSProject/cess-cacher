package test

import (
	"cess-cacher/base/chain"
	"cess-cacher/utils"
	"testing"
	"time"

	"github.com/pkg/errors"
)

// test chain ...
var testCli chain.IChain

func GetTestChainCli() chain.IChain {
	return testCli
}

func InitTestChainClient() error {
	var err error
	//please create chain client with correct account,phrase and chain rpc address
	testCli, err = chain.NewChainClient(
		"ws://172.16.2.243:9944",
		"lunar talent spend shield blade when dumb toilet drastic unique taxi water",
		"cXgZo3RuYkAGhhvCHjAcc9FU13CG44oy8xW6jN39UYvbBaJx5",
		time.Duration(time.Second*15),
	)
	if err != nil {
		return errors.Wrap(err, "init chain client error")
	}
	return nil
}

func TestCacherRegister(t *testing.T) {

	err := InitTestChainClient()
	if err != nil {
		t.Fatal("init chain client error", err)
	}
	ip, err := utils.GetExternalIp()
	if err != nil {
		t.Fatal("get cacher external ip error", err)
	}
	txhash, err := GetTestChainCli().Register(ip, "8080", 1000)
	if err != nil {
		t.Fatal("cacher register error", err)
	}
	t.Log("cacher register tx hash is", txhash)
}

func TestCacherUpdate(t *testing.T) {
	err := InitTestChainClient()
	if err != nil {
		t.Fatal("init chain client error", err)
	}
	ip, err := utils.GetExternalIp()
	if err != nil {
		t.Fatal("get cacher external ip error", err)
	}
	txhash, err := GetTestChainCli().Update(ip, "8081", 1001)
	if err != nil {
		t.Fatal("cacher update error", err)
	}
	t.Log("cacher update tx hash is", txhash)
}

func TestCacherLogout(t *testing.T) {
	err := InitTestChainClient()
	if err != nil {
		t.Fatal("init chain client error", err)
	}
	txhash, err := GetTestChainCli().Logout()
	if err != nil {
		t.Fatal("cacher logout error", err)
	}
	t.Log("cacher logout tx hash is", txhash)
}
