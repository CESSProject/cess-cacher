package cache

import (
	"cess-cacher/logger"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/v3/cpu"
)

type DiskStats struct {
	Total     int64   `json:"total"`
	Used      int64   `json:"used"`
	Available int64   `json:"available"`
	UseRate   float32 `json:"useRate"`
}

type MemoryStats struct {
	Total     int64 `json:"total"`
	Free      int64 `json:"free"`
	Available int64 `json:"available"`
}

type CPUStats struct {
	Num      int     `json:"cpuNum"`
	LoadAvgs float32 `json:"loadAvgs"`
}

type NetStats struct {
	Download int64 `json:"cacheSpeed"`
	Upload   int64 `json:"downloadSpeed"`
}

type CacheStats struct {
	once     sync.Once
	hits     *uint64
	misses   *uint64
	errs     *uint64
	respTime *int64
}

type Stat struct {
	HitRate  float32
	MissRate float32
	ErrRate  float32
}

const FLASH_TIME = time.Hour * 3

var (
	netInfo NetStats
	rwLock  sync.RWMutex //Ensure the synchronization of netInfo
	cstat   *CacheStats
)

func (s *CacheStats) Hit(c uint64) {
	atomic.AddUint64(s.hits, c)
}

func (s *CacheStats) Miss(c uint64) {
	atomic.AddUint64(s.misses, c)
}

func (s *CacheStats) Error(c uint64) {
	atomic.AddUint64(s.errs, c)
}

func (s *CacheStats) UpdateResponseTime(d int64) bool {
	s.once.Do(func() {
		atomic.AddInt64(s.respTime, d)
	})
	return atomic.CompareAndSwapInt64(s.respTime, *s.respTime, (*s.respTime+d)/2)
}

func (s CacheStats) GetResponseTime() int64 {
	return atomic.LoadInt64(s.respTime)
}

func (s CacheStats) GetCacheStats() Stat {
	var stat Stat
	h := atomic.LoadUint64(s.hits)
	m := atomic.LoadUint64(s.misses)
	e := atomic.LoadUint64(s.errs)
	total := float32(h + m + e)
	if total != 0 {
		stat.HitRate = float32(h) / total
		stat.MissRate = float32(m) / total
		stat.ErrRate = float32(e) / total
	}
	return stat
}

func GetDiskStats() (DiskStats, error) {
	var stats DiskStats
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "/opt/"
	}
	out, err := exec.Command("df", pwd).Output()
	if err != nil {
		logger.Uld.Sugar().Errorf("get disk stats error:%v.\n", err)
		return stats, errors.Wrap(err, "get disk stats error")
	}
	tmp := strings.Fields(strings.Split(string(out), "\n")[1])
	tmp[len(tmp)-2] = strings.Replace(tmp[len(tmp)-2], "%", "", 1)
	data := []int64{}
	for i := 1; i < len(tmp)-1; i++ {
		d, err := strconv.ParseInt(tmp[i], 10, 64)
		if err != nil {
			return stats, errors.Wrap(err, "get disk stat error")
		}
		data = append(data, d)
	}
	stats.Total = data[0]
	stats.Used = data[1]
	stats.Available = data[2]
	stats.UseRate = float32(data[3]) / 100
	return stats, nil
}

func GetMemoryStats() (MemoryStats, error) {
	var stats MemoryStats
	out, err := exec.Command("free").Output()
	if err != nil {
		logger.Uld.Sugar().Errorf("get memory stats error:%v.\n", err)
		return stats, errors.Wrap(err, "get memory stats error")
	}
	tmp := strings.Fields(strings.Split(string(out), "\n")[1])[1:]
	tmp = []string{tmp[0], tmp[2], tmp[len(tmp)-1]}
	data := make([]int64, len(tmp))
	for i, v := range tmp {
		d, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return stats, errors.Wrap(err, "get memory stats error")
		}
		data[i] = d
	}
	stats.Total = data[0]
	stats.Free = data[1]
	stats.Available = data[2]
	return stats, nil
}

func GetCPUStats() (CPUStats, error) {
	var stats CPUStats
	stats.Num = runtime.NumCPU()
	rate, err := cpu.Percent(time.Second, false)
	if err != nil {
		return stats, errors.Wrap(err, "get cpu stats error")
	}
	stats.LoadAvgs = float32(math.Trunc(rate[0]*100) / 100)
	return stats, nil
}

func GetNetStats() (NetStats, error) {
	var stats NetStats
	out, err := exec.Command("speedtest", "--json").Output()
	if err != nil {
		return stats, errors.Wrap(err, "get net stats error")
	}
	var data map[string]any
	err = json.Unmarshal(out, &data)
	if err != nil {
		return stats, errors.Wrap(err, "get net stats error")
	}
	stats.Download = int64(data["download"].(float64))
	stats.Upload = int64(data["upload"].(float64))
	return stats, nil
}

func GetNetInfo() NetStats {
	rwLock.RLock()
	defer rwLock.RUnlock()
	return netInfo
}

func UpdateNetStats() {
	var err error
	rwLock.Lock()
	for {
		if netInfo, err = GetNetStats(); err == nil {
			break
		}
		logger.Uld.Sugar().Errorf("get net stats error:%v.\n", err)
	}
	rwLock.Unlock()
	for range time.NewTicker(FLASH_TIME).C {
		stat, err := GetNetStats()
		if err != nil {
			logger.Uld.Sugar().Errorf("get net stats error:%v.\n", err)
			continue
		}
		rwLock.Lock()
		netInfo = stat
		rwLock.Unlock()
	}
}
