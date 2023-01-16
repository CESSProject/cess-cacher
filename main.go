package main

import (
	"cess-cacher/base/cache"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/server"
	"log"
	"os"
)

func main() {
	var configPath string
	logger.InitLogger()
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	if err := config.InitConfig(configPath); err != nil {
		logger.Uld.Sugar().Errorf("init config error:%v", err)
		log.Fatalf("init config error:%v.\n", err)
	}
	if err := cache.InitCache(config.GetConfig()); err != nil {
		logger.Uld.Sugar().Errorf("init cache error:%v", err)
		log.Fatalf("init cache error:%v.\n", err)
	}
	server.SetupGinServer()
}
