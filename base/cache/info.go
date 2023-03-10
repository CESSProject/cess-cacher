package cache

import (
	"cess-cacher/logger"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path"
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
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	UseRate   float32 `json:"useRate"`
}

type MemoryStats struct {
	Total     uint64 `json:"total"`
	Free      uint64 `json:"free"`
	Available uint64 `json:"available"`
}

type CPUStats struct {
	Num      int     `json:"cpuNum"`
	LoadAvgs float32 `json:"loadAvgs"`
}

type NetStats struct {
	Download uint64 `json:"cacheSpeed"`
	Upload   uint64 `json:"downloadSpeed"`
}

type CacheStats struct {
	hits   *uint64
	misses *uint64
	errs   *uint64
}

type Stat struct {
	HitRate  float32 `json:"hitRate"`
	MissRate float32 `json:"missRate"`
	ErrRate  float32 `json:"errRate"`
}

const FLASH_TIME = time.Minute

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
	data := []uint64{}
	for i := 1; i < len(tmp)-1; i++ {
		d, err := strconv.ParseInt(tmp[i], 10, 64)
		if err != nil {
			return stats, errors.Wrap(err, "get disk stat error")
		}
		data = append(data, uint64(d))
	}
	stats.Total = data[0] * 1024
	stats.Used = data[1] * 1024
	stats.Available = data[2] * 1024
	stats.UseRate = float32(data[3]) / 100
	return stats, nil
}

func GetCacheDiskStats() DiskStats {
	used := GetCacheHandle().TotalSize()
	ur := math.Trunc(float64(used)/float64(MaxCacheSize)*100) / 100
	var available uint64
	if MaxCacheSize > used {
		available = MaxCacheSize - used
	}
	return DiskStats{
		Total:     MaxCacheSize,
		Used:      used,
		Available: available,
		UseRate:   float32(ur),
	}
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
	data := make([]uint64, len(tmp))
	for i, v := range tmp {
		d, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return stats, errors.Wrap(err, "get memory stats error")
		}
		data[i] = uint64(d)
	}
	stats.Total = data[0] * 1024
	stats.Free = data[1] * 1024
	stats.Available = data[2] * 1024
	return stats, nil
}

func GetCPUStats() (CPUStats, error) {
	var stats CPUStats
	stats.Num = runtime.NumCPU()
	rate, err := cpu.Percent(time.Second, false)
	if err != nil {
		return stats, errors.Wrap(err, "get cpu stats error")
	}
	stats.LoadAvgs = float32(math.Trunc(rate[0]*10) / 1000)
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
	stats.Download = uint64(data["download"].(float64))
	stats.Upload = uint64(data["upload"].(float64))
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

func DownloadProgressBar(fhash, shash string, size uint64) (float64, int64) {
	fpath := path.Join(FilesDir, fhash, shash)
	if f, err := os.Stat(fpath); err != nil {
		return 0, int64(size) / int64(GetNetInfo().Upload+1)
	} else {
		//download progress
		progress := float64(f.Size()) / float64(size+1)
		//estimated completion time
		ect := int64(size-uint64(f.Size())) / int64(GetNetInfo().Upload+1)
		return progress, ect
	}
}
