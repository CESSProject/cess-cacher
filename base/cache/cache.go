package cache

import (
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/utils"
	"os"
	"path"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

var FilesDir = "./cache/files"
var FilePath = "./cache/metadata.json"
var handler CacheHandle

type ICache interface {
	UpdateResponseTime(d int64) bool
	GetResponseTime() int64
	GetCacheStats() Stat
	FindHashs(hash ...string) []string
	GetHashList() []string
	TotalSize() uint64
	QueryFile(hash string) (FileInfo, bool)
	HitOrLoad(hash string) (bool, error)
	GetFileDir() string
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
		hits:     new(uint64),
		misses:   new(uint64),
		errs:     new(uint64),
		respTime: new(int64),
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
	dir := path.Join(FilesDir, hash)
	if f, err := os.Stat(dir); err == nil {
		fmeta, err := chain.GetChainCli().GetFileMetaInfo(hash)
		if err != nil {
			logger.Uld.Sugar().Errorf("check file %s error:%v", hash, err)
			if err.Error() != chain.ERR_Empty {
				handler.Error(1)
			}
			return false, errors.Wrap(err, "check file error")
		}
		num, err := utils.GetFileNum(dir)
		if err != nil {
			return false, errors.Wrap(err, "check file error")
		}
		if fmeta.Size == types.U64(f.Size()) &&
			num >= (len(fmeta.BlockInfo)-len(fmeta.BlockInfo)/3) {
			handler.LoadInCache(hash, num, uint64(f.Size()))
			return false, nil
		}
	}
	handler.cacheQueue.Insert(hash)
	return true, nil
}

func CheckBadFileAndDel(fid string) bool {
	dir := path.Join(FilesDir, fid)
	if _, err := os.Stat(dir); err != nil {
		return true
	}
	fmeta, err := chain.GetChainCli().GetFileMetaInfo(fid)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			os.Remove(dir)
		}
		return true
	}
	if fs, err := os.ReadDir(dir); err != nil {
		return true
	} else if len(fs) < (len(fmeta.BlockInfo) - len(fmeta.BlockInfo)/3) {
		os.Remove(dir)
		return true
	}
	return false
}
