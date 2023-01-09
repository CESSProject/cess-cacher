package cache

import (
	"cess-cacher/logger"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var FilePath = "./cache/metadata.json"

const FLASH_FILE_TIME = time.Minute

type FileInfo struct {
	Size        int64
	Num         int
	LoadTime    time.Time
	UsedCount   int
	LastAccTime time.Time
}

var (
	hashMap sync.Map
	once    sync.Once
)

func GetHashList() []string {
	var list []string
	hashMap.Range(func(key, value any) bool {
		list = append(list, key.(string))
		return true
	})
	return list
}

func FindHashs(hash []string) []string {
	var res []string
	for _, h := range hash {
		if _, ok := hashMap.Load(h); ok {
			res = append(res, h)
		}
	}
	return res
}

func LoadInCache(hash string, num int, size int64) {
	info := FileInfo{
		Size:        size,
		Num:         num,
		LoadTime:    time.Now(),
		UsedCount:   1,
		LastAccTime: time.Now(),
	}
	hashMap.Store(hash, info)
}

func LoadMetadata() {
	once.Do(func() {
		var list map[string]FileInfo
		bytes, err := os.ReadFile(FilePath)
		if err != nil {
			logger.Uld.Sugar().Errorf("read metadata file error:%v.\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(bytes, &list)
		if err != nil {
			logger.Uld.Sugar().Errorf("unmarshal metadata file error:%v.\n", err)
			os.Exit(1)
		}
		for k, v := range list {
			hashMap.Store(k, v)
		}
	})
}

func SaveMetadata() error {
	var list map[string]FileInfo
	hashMap.Range(func(key, value any) bool {
		list[key.(string)] = value.(FileInfo)
		return true
	})
	bytes, err := json.Marshal(list)
	if err != nil {
		return errors.Wrap(err, "save hash list error")
	}
	err = os.WriteFile(FilePath, bytes, os.ModePerm)
	return errors.Wrap(err, "save hash list error")
}

func FlashMetadataFile() {
	ticker := time.NewTicker(FLASH_FILE_TIME)
	defer ticker.Stop()
	for range ticker.C {
		if err := SaveMetadata(); err != nil {
			logger.Uld.Sugar().Errorf("save metadata file error:%v.\n", err)
		}
	}
}
