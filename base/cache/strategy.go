package cache

import (
	"cess-cacher/logger"
	"math/rand"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
)

var (
	MaxCacheRate        = 0.95
	Threshold           = 0.8
	FreqWeight          = 0.3
	MaxCacheSize uint64 = 0
)

type Item struct {
	Hash     string
	Size     uint64
	Count    int
	Interval time.Duration
}

type LruQueue []Item

func (q LruQueue) Len() int           { return len(q) }
func (q LruQueue) Less(i, j int) bool { return q[i].Interval > q[j].Interval }
func (q LruQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func GetRandomList(c *Cache, pickSize uint64) []Item {
	var (
		size     uint64
		randList []Item
	)
	if pickSize <= 0 || pickSize >= c.TotalSize() {
		pickSize = c.TotalSize()
	}
	r := int(pickSize * 100 / c.TotalSize())
	if r <= 0 {
		r = 50
	}
	check := make(map[string]struct{})
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

func RandomLRU(c *Cache, cleanSize uint64) {
	lruq := LruQueue(GetRandomList(c, cleanSize*3))
	//Access frequency affects the elimination result
	for _, v := range lruq {
		freqW := int64(float64(v.Interval)*FreqWeight) / int64(v.Count)
		RecW := int64(float64(v.Interval) * (1 - FreqWeight))
		v.Interval = time.Duration(RecW + freqW)
	}
	//
	sort.Sort(lruq)
	var size uint64
	for _, v := range lruq {
		if size >= cleanSize {
			break
		}
		size += v.Size
		c.Delete(v.Hash)
		c.delQueue.Insert(v.Hash)
	}
}

func StrategyServer(c *Cache) {
	ticker := time.NewTicker(time.Duration(FLASH_TIME))
	defer ticker.Stop()
	for range ticker.C {
		used := c.TotalSize()
		if used >= uint64(float64(MaxCacheSize)*MaxCacheRate) {
			logger.Uld.Sugar().Info("cache strategy working...")
			RandomLRU(c, used-uint64(float64(MaxCacheSize)*Threshold))
		}
	}
}

func Reorganizate(c *Cache) error {
	dirs, err := os.ReadDir(FilesDir)
	if err != nil {
		return errors.Wrap(err, "reorganizate cache error")
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		df, err := os.ReadDir(path.Join(FilesDir, dir.Name()))
		if err != nil {
			return errors.Wrap(err, "reorganizate cache error")
		}
		for _, f := range df {
			if _, ok := c.hashMap.Load(dir.Name() + "-" + f.Name()); f.IsDir() || ok {
				continue
			}
			if CheckBadFileAndDel(dir.Name(), f.Name()) {
				continue
			}
			var size uint64
			if i, err := f.Info(); err == nil {
				size = uint64(i.Size())
			}
			c.LoadInCache(dir.Name()+"-"+f.Name(), size)
		}
	}
	return nil
}

func CleanCacheServer(c *Cache) {
	for h := range c.delQueue.GetQueue() {
		hash := h
		if _, ok := c.hashMap.Load(hash); ok {
			continue
		}
		paths := strings.Split(hash, "-")
		err := ants.Submit(func() {
			if err := os.Remove(path.Join(FilesDir, paths[0], paths[1])); err != nil {
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
