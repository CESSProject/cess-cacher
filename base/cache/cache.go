package cache

import (
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/utils"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
)

var FilesDir = "./cache/files"
var FilePath = "./cache/metadata.json"
var handler CacheHandle

type ICache interface {
	GetCacheStats() Stat
	FindHashs(hash ...string) []string
	GetHashList() []string
	TotalSize() uint64
	QueryFile(hash string) (FileInfo, bool)
	HitOrLoad(hash string) (bool, error)
	GetFileDir() string
	LoadFailedFile(shash string) (int, bool)
}

type CacheHandle struct {
	*Cache
	*CacheStats
}

func GetCacheHandle() ICache {
	return handler
}

func (h CacheHandle) GetFileDir() string {
	return FilesDir
}

func (h CacheHandle) HitOrLoad(hash string) (bool, error) {
	res := h.FindHashs(hash)
	if len(res) == 1 && res[0] == hash {
		h.Hit(1)
		return true, nil
	}
	//Reduce the impact of invalid requests on hit rate
	if handler.cacheQueue.Query(hash) {
		return false, nil
	}
	downloading, err := CheckAndCacheFile(hash)
	if downloading {
		h.Miss(1)
	}
	if err != nil {
		return false, errors.Wrap(err, "hit cache failed")
	}
	return !downloading, nil
}

func InitCache(conf config.Config) error {
	if conf.CacheDir != "" {
		FilesDir = path.Join(conf.CacheDir, "files")
		FilePath = path.Join(conf.CacheDir, "metadata.json")
	}
	if _, err := os.Stat(FilesDir); err != nil {
		if err = os.MkdirAll(FilesDir, 0755); err != nil {
			return errors.Wrap(err, "init cache error")
		}
	}
	if _, err := os.Stat(FilePath); err != nil {
		f, err := os.Create(FilePath)
		if err != nil {
			return errors.Wrap(err, "init cache error")
		}
		f.WriteString("{}")
		f.Close()
	}
	initMinerInfo()

	stat, err := GetDiskStats()
	if err != nil {
		return errors.Wrap(err, "init strategy error")
	}
	usedSize, err := utils.GetDirSize(FilesDir)
	if err != nil {
		return errors.Wrap(err, "init strategy error")
	}
	MaxCacheSize = stat.Available - usedSize
	qlen := MaxCacheSize / (512 * 1024 * 1024)
	handler = CacheHandle{
		Cache:      NewCache(int(qlen)),
		CacheStats: cstat,
	}
	go handler.Cache.FlashMetadataFile()
	go handler.Cache.CacheFileServer()
	return errors.Wrap(initStrategy(conf, handler.Cache), "init cache error")
}

func initMinerInfo() {
	//init and update net info
	go UpdateNetStats()
	cstat = &CacheStats{
		hits:   new(uint64),
		misses: new(uint64),
		errs:   new(uint64),
	}
}

func initStrategy(conf config.Config, c *Cache) error {
	if utils.IsRateValue(conf.FreqWeight) {
		FreqWeight = conf.FreqWeight
	}
	if utils.IsRateValue(conf.Threshold) {
		Threshold = conf.Threshold
	}
	if utils.IsRateValue(conf.MaxCacheRate) {
		MaxCacheRate = conf.MaxCacheRate
	}
	go CleanCacheServer(c)
	go StrategyServer(c)
	return errors.Wrap(Reorganizate(c), "init strategy error")
}

func CheckAndCacheFile(hash string) (bool, error) {
	paths := strings.Split(hash, "-")
	file := path.Join(FilesDir, paths[0], paths[1])
	if f, err := os.Stat(file); err == nil {
		fmeta, err := chain.GetChainCli().GetFileMetaInfo(paths[0])
		if err != nil {
			logger.Uld.Sugar().Errorf("check file %s error:%v", hash, err)
			if err.Error() != chain.ERR_Empty {
				handler.Error(1)
			}
			return false, errors.Wrap(err, "check file error")
		}
		var size int64
		for _, block := range fmeta.BlockInfo {
			if string(block.BlockId[:]) == paths[1] {
				size = int64(block.BlockSize)
				break
			}
		}
		if size == f.Size() {
			handler.LoadInCache(hash, uint64(f.Size()))
			return false, nil
		}
	}
	handler.cacheQueue.Insert(hash)
	return true, nil
}

func CheckBadFileAndDel(fid, sid string) bool {
	file := path.Join(FilesDir, fid, sid)
	var size int64
	if f, err := os.Stat(file); err != nil {
		return true
	} else {
		size = f.Size()
	}
	fmeta, err := chain.GetChainCli().GetFileMetaInfo(fid)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			os.Remove(file)
		}
		return true
	}
	for _, block := range fmeta.BlockInfo {
		if string(block.BlockId[:]) != sid {
			continue
		}
		if int64(block.BlockSize) != size {
			os.Remove(file)
			return true
		}
	}
	return false
}
