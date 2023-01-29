package cmd

import (
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/utils"
	"log"

	"github.com/spf13/cobra"
)

func RegisterCacher() {
	ip, err := utils.GetExternalIp()
	if err != nil {
		log.Fatalf("register cacher error %v", err)
	}
	cli := chain.GetChainCli()
	conf := config.GetConfig()
	txhash, err := cli.Register(ip, conf.ServerPort, conf.BytePrice)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("register cacher success,tx hash is ", txhash)
}

func UpdateCacherInfo() {
	ip, err := utils.GetExternalIp()
	if err != nil {
		log.Fatalf("update cacher info error %v", err)
	}
	cli := chain.GetChainCli()
	conf := config.GetConfig()
	txhash, err := cli.Update(ip, conf.ServerPort, conf.BytePrice)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("update cacher info success,tx hash is ", txhash)
}

func LogoutCacher() {
	txhash, err := chain.GetChainCli().Logout()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("logout cacher success,tx hash is ", txhash)
}

func BuildProfile(cmd *cobra.Command) {
	var configPath string
	logger.InitLogger()
	if path, _ := cmd.Flags().GetString("c"); path != "" {
		configPath = path
	} else {
		configPath, _ = cmd.Flags().GetString("config")
	}
	if err := config.InitConfig(configPath); err != nil {
		logger.Uld.Sugar().Errorf("init config error:%v", err)
		log.Fatalf("init config error:%v.\n", err)
	}
}
