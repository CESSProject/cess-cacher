package cmd

import (
	"cess-cacher/base/cache"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/server"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cacher",
	Short: "CESS CDN cache miner",
}

func Execute() {
	rootCmd.AddCommand(
		Command_RegisterCacher(),
		Command_UpdateCacherInfo(),
		Command_LogoutCacher(),
		Command_RunCacheServer(),
	)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func Command_RegisterCacher() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "register cacher on CESS chain",
		Run: func(cmd *cobra.Command, args []string) {
			BuildProfile(cmd)
			RegisterCacher()
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
}

func Command_UpdateCacherInfo() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "update cacher info on CESS chain",
		Run: func(cmd *cobra.Command, args []string) {
			BuildProfile(cmd)
			UpdateCacherInfo()
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
}

func Command_LogoutCacher() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "logout cacher from CESS chain",
		Run: func(cmd *cobra.Command, args []string) {
			BuildProfile(cmd)
			LogoutCacher()
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
}

func Command_RunCacheServer() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "run cache server",
		Run: func(cmd *cobra.Command, args []string) {
			BuildProfile(cmd)
			if err := cache.InitCache(config.GetConfig()); err != nil {
				logger.Uld.Sugar().Errorf("init cache error:%v", err)
				log.Fatalf("init cache error:%v.\n", err)
			}
			server.SetupGinServer()
		},
		DisableFlagsInUseLine: true,
	}
}
