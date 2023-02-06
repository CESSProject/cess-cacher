package service

import (
	"cess-cacher/base/cache"
	"cess-cacher/config"
	resp "cess-cacher/server/response"
	"cess-cacher/utils"
	"os"
	"path"

	"github.com/pkg/errors"
)

type MinerStats struct {
	GeoLocation string            `json:"geoLocation"`
	BytePrice   uint64            `json:"bytePrice"`
	MinerStatus string            `json:"status"`
	NetStats    cache.NetStats    `json:"netStats"`
	MemoryStats cache.MemoryStats `json:"memStats"`
	CPUStats    cache.CPUStats    `json:"cpuStats"`
	DiskStats   cache.DiskStats   `json:"diskStats"`
	CacheStat   cache.Stat        `json:"cacheStat"`
}

type FileStat struct {
	Cached     bool     `json:"cached"`
	Price      uint64   `json:"price"`
	Size       uint64   `json:"size"`
	ShardCount int      `josn:"shardCount"`
	Shards     []string `json:"shards"`
}

func QueryMinerStats() (MinerStats, resp.Error) {
	var (
		mstat MinerStats
		err   error
	)
	mstat.MinerStatus = "active"
	mstat.NetStats = cache.GetNetInfo()
	mstat.CacheStat = cache.GetCacheHandle().GetCacheStats()
	mstat.BytePrice = config.GetConfig().BytePrice
	mstat.MemoryStats, err = cache.GetMemoryStats()
	if err != nil {
		return mstat, resp.NewError(500, errors.Wrap(err, "query miner stats error"))
	}
	mstat.CPUStats, err = cache.GetCPUStats()
	if err != nil {
		return mstat, resp.NewError(500, errors.Wrap(err, "query miner stats error"))
	}
	mstat.DiskStats, err = cache.GetDiskStats()
	if err != nil {
		return mstat, resp.NewError(500, errors.Wrap(err, "query miner stats error"))
	}
	extIp, err := utils.GetExternalIp()
	if err != nil {
		return mstat, resp.NewError(500, errors.Wrap(err, "query miner stats error"))
	}
	country, city, err := utils.ParseCountryFromIp(extIp)
	if err != nil {
		return mstat, resp.NewError(500, errors.Wrap(err, "query miner stats error"))
	}
	mstat.GeoLocation = country + "," + city
	return mstat, nil
}

func QueryCachedFiles() []string {
	return cache.GetCacheHandle().GetHashList()
}

func QueryFileInfo(hash string) FileStat {
	var stat FileStat
	info, ok := cache.GetCacheHandle().QueryFile(hash)
	if !ok {
		//query info from chain
		return stat
	}
	stat.Cached = true
	stat.Price = uint64(info.Size) * config.GetConfig().BytePrice
	stat.Size = uint64(info.Size)
	stat.ShardCount = info.Num
	fs, err := os.ReadDir(path.Join(cache.FilesDir, hash))
	if err != nil {
		return stat
	}
	stat.Shards = make([]string, len(fs))
	for i, v := range fs {
		stat.Shards[i] = v.Name()
	}
	return stat
}

func QueryBytePrice() uint64 {
	return config.GetConfig().BytePrice
}
