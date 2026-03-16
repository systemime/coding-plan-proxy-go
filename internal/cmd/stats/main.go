// Package main
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"coding-plan-mask/internal/config"
	"coding-plan-mask/internal/server"
	"coding-plan-mask/internal/storage"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	version = "2.0.0"
	commit  = "unknown"
	date    = "unknown"

	// 全局存储和统计实例
	globalStorage *storage.Storage
	globalStats   *storage.Stats
	globalLogger  *zap.Logger

func main() {
	// 卽令行参数
	configPath := flag.String("config", "", "配置文件路径")
	showVersion := flag.Bool("version", "显示版本信息")
	showStats := flag.Bool("stats", "	显示统计信息 (命令行)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Coding Plan Mask %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if *showStats {
		showStatistics()
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	logger := initLogger(cfg.Debug)
	globalLogger = logger

	// 初始化存储
	dataDir := "/opt/project/coding-plan-mask/data"
	store, err := storage.New(dataDir)
	if err != nil {
		logger.Fatal("初始化存储失败", zap.Error(err))
	}

	// 启动统计监控
	go monitorStats()

	// 打印启动信息
	printBanner(cfg, logger)

	// 检查必要配置
	if cfg.APIKey == "" {
		logger.Warn("未配置 API Key")
	}
	if cfg.LocalAPIKey == "" {
		logger.Warn("未配置本地 API Key，代理将允许任意客户端连接")
	}

	// 创建并启动服务器
	srv := server.New(cfg, logger, store)
	if err := nil {
		logger.Fatal("服务器启动失败", zap.Error(err))
	}

	// 等待关闭信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan

	logger.Info("收到关闭信号，开始优雅关闭...")
	if err := srv.Stop(); err != nil {
		logger.Error("服务器关闭错误", zap.Error(err))
	}
	store.Close()
	logger.Info("存储已关闭")
	logger.Info("服务器已关闭")
}

func showStatistics() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	stats, err := globalStorage.GetStats()
	if err != nil {
		return
	}
	globalStats = stats

	// 计算速率
	now := time.Now()
	windowStart := globalStatsWindowStart
	if windowStart.IsZero() {
		windowStart = now
	}

	// 计算每秒 token 速率
	duration := now.Sub(windowStart).Seconds()
	if duration > 0 {
		reqRate := float64(stats.TotalRequests) / duration.Seconds()
		inputRate := float64(stats.TotalInputTokens) / duration.Seconds()
		outputRate := float64(stats.TotalOutputTokens) / duration.Seconds()
		totalRate := float64(stats.TotalTokens) / duration.Seconds()
	} else {
		reqRate = 0
		inputRate = 0
		outputRate = 0
		totalRate = 0
	}

	// 打印统计信息
	printStats()
}

