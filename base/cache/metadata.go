package cache

import (
	"cess-cacher/base/trans"
	"cess-cacher/logger"
	"cess-cacher/utils"
	"encoding/json"
	"os"
	"path"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
)

const FLASH_FILE_TIME = time.Minute
const DEFAULT_QUEUE_SIZE = 512

type FileInfo struct {
	Size        int64
	Num         int
	LoadTime    time.Time
	UsedCount   int
	LastAccTime time.Time
}

type HashQueue struct {
	rw     sync.RWMutex
	queue  chan string
	filter map[string]struct{}
}

type Cache struct {
	rw         sync.RWMutex
	hashMap    sync.Map
	size       int64
	delQueue   *HashQueue
	cacheQueue *HashQueue
}

func NewCache(qlen int) *Cache {
	cache := &Cache{
		delQueue:   NewQueue(qlen),
		cacheQueue: NewQueue(qlen),
	}
	cache.LoadMetadata()
	return cache
}

func (c *Cache) TotalSize() int64 {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.size
}

func (c *Cache) GetHashList() []string {
	var list []string
	c.hashMap.Range(func(key, value any) bool {
		list = append(list, key.(string))
		return true
	})
	return list
}

func (c *Cache) FindHashs(hash ...string) []string {
	var res []string
	for _, h := range hash {
		if _, ok := c.hashMap.Load(h); ok {
			res = append(res, h)
		}
	}
	return res
}

func (c *Cache) QueryFile(hash string) (FileInfo, bool) {
	if info, ok := c.hashMap.Load(hash); !ok {
		return FileInfo{}, ok
	} else {
		return info.(FileInfo), ok
	}
}

func (c *Cache) LoadInCache(hash string, num int, size int64) {
	if num <= 0 || size <= 0 {
		return
	}
	info := FileInfo{
		Size:        size,
		Num:         num,
		LoadTime:    time.Now(),
		UsedCount:   1,
		LastAccTime: time.Now(),
	}
	v, ok := c.hashMap.Load(hash)
	if !ok || v.(FileInfo).Size != size {
		c.rw.Lock()
		c.size = c.size - v.(FileInfo).Size + size
		c.rw.Unlock()
	}
	c.hashMap.Store(hash, info)
}

func NewQueue(size int) *HashQueue {
	if size <= 0 {
		size = DEFAULT_QUEUE_SIZE
	}
	return &HashQueue{
		queue:  make(chan string, size),
		filter: make(map[string]struct{}),
	}
}

func (q *HashQueue) GetQueue() <-chan string {
	return q.queue
}

func (q *HashQueue) Insert(hash string) {
	q.rw.Lock()
	if _, ok := q.filter[hash]; !ok {
		q.filter[hash] = struct{}{}
		q.rw.Unlock()
		q.queue <- hash
	} else {
		q.rw.Unlock()
	}
}

func (q *HashQueue) Delete(hash string) {
	q.rw.Lock()
	delete(q.filter, hash)
	q.rw.Unlock()
}

func (q *HashQueue) Query(hash string) bool {
	q.rw.RLock()
	defer q.rw.RUnlock()
	_, ok := q.filter[hash]
	return ok
}

func (c *Cache) LoadMetadata() {
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
	var size int64
	for k, v := range list {
		c.hashMap.Store(k, v)
		size += v.Size
	}
	c.rw.Lock()
	c.size = size
	c.rw.Unlock()
}

func (c *Cache) SaveMetadata() error {
	var list map[string]FileInfo
	c.hashMap.Range(func(key, value any) bool {
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

func (c *Cache) FlashMetadataFile() {
	ticker := time.NewTicker(FLASH_FILE_TIME)
	defer ticker.Stop()
	for range ticker.C {
		if err := c.SaveMetadata(); err != nil {
			logger.Uld.Sugar().Errorf("save metadata file error:%v.\n", err)
		}
	}
}

func (c *Cache) CacheFileServer() {
	for h := range c.cacheQueue.GetQueue() {
		hash := h
		dir := path.Join(FilesDir, hash)
		if _, err := os.Stat(dir); err != nil {
			if err = os.Mkdir(dir, 0755); err != nil {
				continue
			}
			ants.Submit(func() {
				err := trans.DownloadFile(hash, dir)
				if err != nil {
					logger.Uld.Sugar().Errorf("download file %s from storage error:%v.\n", hash, err)
					return
				}
				num, err := utils.GetFileNum(dir)
				if err != nil {
					logger.Uld.Sugar().Errorf("get slice number of file %s error:%v.\n", hash, err)
					return
				}
				size, err := utils.GetDirSize(dir)
				if err != nil {
					logger.Uld.Sugar().Errorf("get size of file %s error:%v.\n", hash, err)
					return
				}
				c.LoadInCache(hash, num, size)
			})
		}
	}
}
