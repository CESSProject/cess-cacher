package service

import (
	"cess-cacher/base/cache"
	resp "cess-cacher/server/response"
	"cess-cacher/utils"

	"github.com/pkg/errors"
)

type MinerStats struct {
	GeoLocation string            `json:"geoLocation"`
	BytePrice   uint              `json:"bytePrice"`
	MinerStatus string            `json:"status"`
	NetStats    cache.NetStats    `json:"netStats"`
	MemoryStats cache.MemoryStats `json:"memStats"`
	CPUStats    cache.CPUStats    `json:"cpuStats"`
	DiskStats   cache.DiskStats   `json:"diskStats"`
}

type FileStat struct {
	Price      uint   `json:"price"`
	Size       uint64 `json:"size"`
	ShardCount int    `josn:"shardCount"`
}

func QueryMinerStats() (MinerStats, resp.Error) {
	var (
		mstat MinerStats
		err   error
	)
	mstat.MinerStatus = "active"
	mstat.NetStats = cache.GetNetInfo()
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
		return stat
	}
	stat.Price = 100
	stat.Size = uint64(info.Size)
	stat.ShardCount = info.Num
	return stat
}
