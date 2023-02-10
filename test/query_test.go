package test

import (
	"cess-cacher/base/cache"
	"cess-cacher/config"
	"cess-cacher/utils"
	"encoding/json"
	"testing"
)

func TestQueryDiskStat(t *testing.T) {
	err := config.InitConfig("../config/config.toml")
	if err != nil {
		t.Fatal("init config error", err)
	}

	err = cache.InitCache(config.GetConfig())
	if err != nil {
		t.Fatal("init cache error", err)
	}
	machineDisk, err := cache.GetDiskStats()
	if err != nil {
		t.Fatal("query disk stats error", err)
	}
	bytes, err := json.Marshal(machineDisk)
	if err != nil {
		t.Fatal("marshal cacher machine disk error", err)
	}
	t.Log("cacher machine disk stat", string(bytes))

	cacheDisk := cache.GetCacheDiskStats()
	bytes, err = json.Marshal(cacheDisk)
	if err != nil {
		t.Fatal("marshal cacher logic disk error", err)
	}
	t.Log("cacher logic disk stat", string(bytes))
}

// get net stat need long time ,please use 'go test -v --run TestQueryMachineStat --timeout=2m'
func TestQueryMachineStat(t *testing.T) {
	cpuStat, err := cache.GetCPUStats()
	if err != nil {
		t.Fatal("query cpu stat error", err)
	}
	bytes, err := json.Marshal(cpuStat)
	if err != nil {
		t.Fatal("marshal cpu stat error", err)
	}
	t.Log("cacher cpu stat", string(bytes))

	memStat, err := cache.GetMemoryStats()
	if err != nil {
		t.Fatal("query memory stat error", err)
	}
	bytes, err = json.Marshal(memStat)
	if err != nil {
		t.Fatal("marshal memory stat error", err)
	}
	t.Log("cacher memory stat", string(bytes))

	netStat, err := cache.GetNetStats()
	if err != nil {
		t.Fatal("query net stat error", err)
	}
	bytes, err = json.Marshal(netStat)
	if err != nil {
		t.Fatal("marshal net stat error", err)
	}
	t.Log("cacher net stat", string(bytes))
}

func TestGetLocation(t *testing.T) {
	extraIp, err := utils.GetExternalIp()
	if err != nil {
		t.Fatal("get cacher extranal ip error", err)
	}
	t.Log("cacher extranal ip is", extraIp)
	country, city, err := utils.ParseCountryFromIp(extraIp)
	if err != nil {
		t.Fatal("get cacher location info error", err)
	}
	t.Log("cacher loaction:", country, city)
}
