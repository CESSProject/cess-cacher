package test

import (
	"cess-cacher/base/cache"
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/server"
	"testing"
	"time"
)

func TestInitConfig(t *testing.T) {
	err := config.InitConfig("../config/config.toml")
	if err != nil {
		t.Fatal("init config error", err)
	}
	t.Log("config info", config.GetConfig())

}

func TestInitLogger(t *testing.T) {
	logger.InitLogger()
	logger.Uld.Sugar().Info("run test")
}

func TestInitCache(t *testing.T) {
	err := config.InitConfig("../config/config.toml")
	if err != nil {
		t.Fatal("init config error", err)
	}

	err = cache.InitCache(config.GetConfig())
	if err != nil {
		t.Fatal("init cache error", err)
	}
	cacheHandle := cache.GetCacheHandle()

	stat := cacheHandle.GetCacheStats()
	t.Log("cache stat", stat)

	cachedFiles := cacheHandle.GetHashList()
	t.Log("cached files", cachedFiles)

	cachedSize := cacheHandle.TotalSize()
	t.Log("cached size", cachedSize)
}

func TestInitChainClient(t *testing.T) {
	err := config.InitConfig("../config/config.toml")
	if err != nil {
		t.Fatal("init config error", err)
	}

	err = chain.InitChainClient(config.GetConfig())
	if err != nil {
		t.Fatal("init chain client error", err)
	}
	cli := chain.GetChainCli()
	pubkey := cli.GetPublicKey()
	if err != nil {
		t.Fatal("init chain client error", err)
	}
	t.Log("cacher publick key", pubkey)
}

func TestInitServer(t *testing.T) {
	err := config.InitConfig("../config/config.toml")
	if err != nil {
		t.Fatal("init config error", err)
	}
	go server.SetupGinServer()
	time.Sleep(time.Second * 5)
}
