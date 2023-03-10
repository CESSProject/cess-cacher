package cmd

import (
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"log"

	"github.com/spf13/cobra"
)

func RegisterCacher() {
	cli := chain.GetChainCli()
	conf := config.GetConfig()
	txhash, err := cli.Register(conf.ServerIp, conf.ServerPort, conf.BytePrice)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("register cacher success,tx hash is ", txhash)
}

func UpdateCacherInfo() {
	cli := chain.GetChainCli()
	conf := config.GetConfig()
	txhash, err := cli.Update(conf.ServerIp, conf.ServerPort, conf.BytePrice)
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
	if err := chain.InitChainClient(config.GetConfig()); err != nil {
		logger.Uld.Sugar().Errorf("init chain client error:%v", err)
		log.Fatalf("init chain client error:%v.\n", err)
	}

	//test chain
	if err := chain.InitChainClient(config.GetConfig()); err != nil {
		logger.Uld.Sugar().Errorf("init test chain client error:%v", err)
		log.Fatalf("init test chain client error:%v.\n", err)
	}
}
