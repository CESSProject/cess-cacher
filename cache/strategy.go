package cache

import (
	"cess-cacher/logger"
	"math/rand"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pkg/errors"
)

var FilesDir = "./cache/files"

type Item struct {
	Hash     string
	Size     int64
	Count    int
	Interval time.Duration
}

type LruQueue []Item
type LfuQueue []Item

func (q LruQueue) Len() int           { return len(q) }
func (q LruQueue) Less(i, j int) bool { return q[i].Interval > q[j].Interval }
func (q LruQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func (q LfuQueue) Len() int           { return len(q) }
func (q LfuQueue) Less(i, j int) bool { return q[i].Count < q[j].Count }
func (q LfuQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func GetRandomList(totalSize int64) []Item {
	var (
		size     int64
		check    map[string]struct{}
		randList []Item
	)
	now := time.Now()
	for {
		if size >= int64(float64(totalSize)*0.45) {
			break
		}
		hashMap.Range(func(key, value any) bool {
			if size >= int64(float64(totalSize)*0.45) {
				return false
			}
			k, v := key.(string), value.(FileInfo)
			if _, ok := check[k]; rand.Intn(100) < 48 && !ok {
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

func RandomLRU(totalSize, cleanSize int64) {
	list := GetRandomList(totalSize)
	lruq := LruQueue(list)
	var lfuq LfuQueue
	copy(lfuq, list)
	sort.Sort(lruq)
	sort.Sort(lfuq)
	l := int(float64(len(list)) * 0.45)
	tmp := make(map[string]struct{})
	for i := 0; i < l; i++ {
		tmp[lfuq[i].Hash] = struct{}{}
	}
	var size int64
	cleanList := make(map[string]struct{})
	for i := 0; i < l; i++ {
		if size >= cleanSize {
			break
		}
		if _, ok := tmp[lruq[i].Hash]; ok {
			cleanList[lruq[i].Hash] = struct{}{}
			size += lruq[i].Size
		}
	}
	for _, v := range lruq {
		if size >= cleanSize {
			break
		}
		if _, ok := cleanList[v.Hash]; !ok {
			cleanList[v.Hash] = struct{}{}
			size += v.Size
		}
	}
	for k := range cleanList {
		hashMap.Delete(k)
		err := os.Remove(path.Join(FilesDir, k))
		if err != nil {
			logger.Uld.Sugar().Errorf("reomve cache file %s error:%v.\n", k, err)
		}
	}
}

func RunEliminationServer() {

}

func Reorganization() error {
	dirs, err := os.ReadDir(FilesDir)
	if err != nil {
		return errors.Wrap(err, "reorganizate cache error")
	}
	for _, dir := range dirs {
		if _, ok := hashMap.Load(dir.Name()); !dir.IsDir() || ok {
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
		hashMap.Store(dir.Name(), info)
	}
	return nil
}
