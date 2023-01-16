package cache

import (
	"cess-cacher/logger"
	"math/rand"
	"os"
	"path"
	"sort"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
)

var (
	MaxCacheRate       = 0.95
	Threshold          = 0.8
	FreqWeight         = 0.3
	MaxCacheSize int64 = 0
)

type Item struct {
	Hash     string
	Size     int64
	Count    int
	Interval time.Duration
}

type LruQueue []Item

func (q LruQueue) Len() int           { return len(q) }
func (q LruQueue) Less(i, j int) bool { return q[i].Interval > q[j].Interval }
func (q LruQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func GetRandomList(c *Cache, pickSize int64) []Item {
	var (
		size     int64
		check    map[string]struct{}
		randList []Item
	)
	if pickSize <= 0 || pickSize >= c.TotalSize() {
		return randList
	}
	r := int(pickSize * 100 / c.TotalSize())
	if r <= 0 {
		r = 50
	}
	now := time.Now()
	for size < pickSize {
		c.hashMap.Range(func(key, value any) bool {
			if size >= pickSize {
				return false
			}
			k, v := key.(string), value.(FileInfo)
			if _, ok := check[k]; rand.Intn(100) < r && !ok {
				check[k] = struct{}{}
				randList = append(randList, Item{
					Hash:     k,
					Size:     v.Size,
					Count:    v.UsedCount,
					Interval: now.Sub(v.LastAccTime),
				})
				size += v.Size
			}
			return true
		})
	}
	return randList
}

func RandomLRU(c *Cache, cleanSize int64) {
	lruq := LruQueue(GetRandomList(c, cleanSize*3))
	//Access frequency affects the elimination result
	for _, v := range lruq {
		freqW := int64(float64(v.Interval)*FreqWeight) / int64(v.Count)
		RecW := int64(float64(v.Interval) * (1 - FreqWeight))
		v.Interval = time.Duration(RecW + freqW)
	}
	//
	sort.Sort(lruq)
	var size int64
	for _, v := range lruq {
		if size >= cleanSize {
			break
		}
		size += v.Size
		c.hashMap.Delete(v.Hash)
		c.delQueue.Insert(v.Hash)
	}
}

func StrategyServer(c *Cache) {
	NetInfo := GetNetInfo()
	interval := MaxCacheSize * 3 / 100 / NetInfo.Download * int64(time.Second)
	if interval > int64(FLASH_TIME) || interval <= 0 {
		interval = int64(FLASH_TIME)
	}
	ticker := time.NewTicker(time.Duration(interval))
	defer ticker.Stop()
	for range ticker.C {
		used := c.TotalSize()
		if used >= int64(float64(MaxCacheSize)*MaxCacheRate) {
			RandomLRU(c, used-int64(float64(MaxCacheSize)*Threshold))
		}
	}
}

func Reorganizate(c *Cache) error {
	dirs, err := os.ReadDir(FilesDir)
	if err != nil {
		return errors.Wrap(err, "reorganizate cache error")
	}
	for _, dir := range dirs {
		if CheckBadFileAndDel(dir.Name()) {
			continue
		}
		if _, ok := c.hashMap.Load(dir.Name()); !dir.IsDir() || ok {
			continue
		}
		subDirs, err := os.ReadDir(path.Join(FilesDir, dir.Name()))
		if err != nil {
			logger.Uld.Sugar().Errorf("read dir %s error:%v.\n", dir.Name(), err)
			continue
		}
		info := FileInfo{}
		for _, file := range subDirs {
			if file.IsDir() {
				continue
			}
			if i, err := file.Info(); err == nil {
				info.Size += i.Size()
				info.Num++
			}
		}
		info.LoadTime = time.Now()
		info.LastAccTime = time.Now()
		info.UsedCount = 1
		c.hashMap.Store(dir.Name(), info)
	}
	return nil
}

func CleanCacheServer(c *Cache) {
	for h := range c.delQueue.GetQueue() {
		hash := h
		if _, ok := c.hashMap.Load(hash); ok {
			continue
		}
		err := ants.Submit(func() {
			if err := os.Remove(path.Join(FilesDir, hash)); err != nil {
				logger.Uld.Sugar().Errorf("reomve cache file %s error:%v.\n", hash, err)
				c.delQueue.Insert(hash)
				return
			}
			c.delQueue.Delete(hash)
		})
		if err != nil {
			c.delQueue.Insert(hash)
			logger.Uld.Sugar().Errorf("clean cache file %s error:%v.\n", hash, err)
		}
	}
}
