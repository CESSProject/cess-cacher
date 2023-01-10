package utils

import (
	"log"
	"os"
	"os/exec"
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
	Download int64 `json:"downloadSpeed"`
	Upload   int64 `json:"cacheSpeed"`
}

func GetDiskStats() DiskStats {
	var stats DiskStats
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "/opt/"
	}
	out, err := exec.Command("df", pwd).Output()
	if err != nil {
		//logger.Uld.Sugar().Errorf("get disk stats error:%v.\n", err)
		log.Println("get disk stats error", err)
	}
	log.Println(string(out))
	return stats
}
