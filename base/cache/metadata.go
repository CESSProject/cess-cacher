package cache

import (
	"cess-cacher/base/trans"
	"cess-cacher/logger"
	"encoding/json"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
)

const (
	FLASH_FILE_TIME      = time.Minute
	DEFAULT_QUEUE_SIZE   = 512
	CLEAR_FAILEDMAP_TIME = time.Hour * 6
)

type FileInfo struct {
	Size        uint64
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
	size       uint64
	delQueue   *HashQueue
	cacheQueue *HashQueue
	failedMap  sync.Map
}

func NewCache(qlen int) *Cache {
	cache := &Cache{
		delQueue:   NewQueue(qlen),
		cacheQueue: NewQueue(qlen),
	}
	cache.LoadMetadata()
	return cache
}

func (c *Cache) AddFailedFile(shash string) {
	if count, ok := c.failedMap.LoadOrStore(shash, int(1)); ok {
		c.failedMap.Store(shash, count.(int)+1)
	}
}

func (c *Cache) DelFailedFile(shash string) {
	c.failedMap.Delete(shash)
}

func (c *Cache) LoadFailedFile(shash string) (int, bool) {
	v, ok := c.failedMap.Load(shash)
	if !ok {
		return 0, ok
	}
	return v.(int), ok
}

func (c *Cache) ClearFailedMap(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		c.failedMap.Range(func(key, value any) bool {
			c.failedMap.Delete(key)
			return true
		})
	}

}

func (c *Cache) TotalSize() uint64 {
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

func (c *Cache) LoadInCache(hash string, size uint64) {
	if size <= 0 {
		return
	}
	info := FileInfo{
		Size:        size,
		LoadTime:    time.Now(),
		UsedCount:   1,
		LastAccTime: time.Now(),
	}
	if v, ok := c.hashMap.Load(hash); ok {
		if v.(FileInfo).Size != size {
			c.rw.Lock()
			c.size = c.size - v.(FileInfo).Size + size
			c.rw.Unlock()
		}
	} else {
		c.rw.Lock()
		c.size = c.size + size
		c.rw.Unlock()
	}
	c.hashMap.Store(hash, info)
}

func (c *Cache) Delete(hash string) {
	c.rw.Lock()
	defer c.rw.Unlock()
	v, ok := c.hashMap.LoadAndDelete(hash)
	if ok {
		c.size -= v.(FileInfo).Size
	}
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
	var size uint64
	for k, v := range list {
		paths := strings.Split(k, "-")
		if CheckBadFileAndDel(paths[0], paths[1]) {
			continue
		}
		c.hashMap.Store(k, v)
		size += v.Size
	}
	c.rw.Lock()
	c.size = size
	c.rw.Unlock()
}

func (c *Cache) SaveMetadata() error {
	list := make(map[string]FileInfo)
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
	go c.ClearFailedMap(CLEAR_FAILEDMAP_TIME)
	lockMap := sync.Map{}
	for h := range c.cacheQueue.GetQueue() {
		hash := h
		paths := strings.Split(hash, "-")
		dir := path.Join(FilesDir, paths[0])
		if _, err := os.Stat(dir); err != nil {
			if err = os.Mkdir(dir, 0777); err != nil {
				continue
			}
		}
		if _, ok := lockMap.Load(hash); ok {
			continue
		}
		if _, err := os.Stat(path.Join(dir, paths[1])); err != nil {
			ants.Submit(func() {
				lockMap.Store(hash, struct{}{})
				defer lockMap.Delete(hash)
				err := trans.DownloadFile(paths[0], dir, paths[1])
				if err != nil {
					c.AddFailedFile(paths[1])
					logger.Uld.Sugar().Errorf("download file %s from storage error:%v.\n", hash, err)
					return
				}
				fs, err := os.Stat(path.Join(dir, paths[1]))
				if err != nil {
					c.AddFailedFile(paths[1])
					logger.Uld.Sugar().Errorf("get size of file %s error:%v.\n", hash, err)
					return
				}
				c.LoadInCache(hash, uint64(fs.Size()))
			})
		}
	}
}
