/*
Copyright © 2022 Joker
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	_ "sysafari.com/customs/cguard/docs"
	"sysafari.com/customs/cguard/internal/config"
	"sysafari.com/customs/cguard/internal/database"
	"sysafari.com/customs/cguard/internal/service"
	web "sysafari.com/customs/cguard/internal/web"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cguard",
	Short: "Generate various documents required by the customs declaration system",
	Long: `In order to ensure the normal customs declaration business,
the data files or view files of various report types that need to be generated. For example:
1. LWT documents that need to be submitted when customs declaration encounters inspection.
..
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化参数
		cfg, err := config.InitConfig()
		if err != nil {
			log.Fatalf("初始化配置失败: %v", err)
			return
		}

		// 初始化数据库连接
		err = database.InitDB(&cfg.MySQL)
		if err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}

		// 开启rabbitmq消费者
		if err := config.InitRabbitMQ(); err != nil {
			log.Fatalf("初始化RabbitMQ失败: %v", err)
		}

		// 开启lwt请求消费者
		if err := config.StartLwtRequestConsumer(service.GenerateLWTExcel); err != nil {
			log.Fatalf("启动lwt请求消费者失败: %v", err)
		}

		// 设置优雅关闭
		setupGracefulShutdown()

		// 启动Web服务器
		if err := web.StartServer(); err != nil {
			log.Infof("服务器关闭: %v", err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".cguard.yaml", "config file (default is .cguard.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	fmt.Println("Init viper ...")
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cguard" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cguard")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())

		// Initialize global configuration
		_, err := config.InitConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error initializing config:", err)
		}
	}

	// init logging
	initLogging()
}

// initLogging Initialize logging
func initLogging() {
	path, _ := os.Executable()
	_, exec := filepath.Split(path)
	fmt.Println(exec)

	cfg := config.GetConfig()
	logDir := cfg.Log.LogBase
	logFilename := exec + ".log"

	config.InitLog(logDir, logFilename, cfg.Log.Level)
}

// setupGracefulShutdown 设置优雅关闭机制
func setupGracefulShutdown() {
	// 设置优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		// 等待中断信号
		<-quit
		log.Info("正在关闭服务器...")

		// 优雅关闭 web 服务器（10秒超时）
		if err := web.ShutdownServer(10 * time.Second); err != nil {
			log.Errorf("服务器关闭出错: %v", err)
		}

		// 关闭其他资源
		config.CloseRabbitMQ()
		database.CloseDB()

		log.Info("服务器已关闭")
		os.Exit(0)
	}()
}
