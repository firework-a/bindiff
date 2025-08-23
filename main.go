package main

import (
	"bindiff/cmd"
	"bindiff/pkg/config"
	"bindiff/pkg/logger"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// 全局配置
	cfg *config.Config

	// 命令行选项
	configFile   string
	logLevel     string
	showProgress bool
	verbose      bool
	repoDir      string
	maxWorkers   int
	useParallel  bool
	enableFFT    bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "bdiff",
		Short: "Enhanced binary diff and patch tool",
		Long: `bdiff - An advanced binary delta compressor and patcher using:
- Hash-based block matching for efficient diffing
- FFT-based alignment optimization
- Parallel processing for large files
- Progress tracking and detailed logging
- Configurable compression levels`,
		PersistentPreRunE: initializeApp,
		PersistentPostRun: cleanupApp,
	}

	// 全局选项
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Config file path")
	rootCmd.PersistentFlags().StringVarP(&repoDir, "repo", "r", ".bindiff", "Repository directory")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVarP(&showProgress, "progress", "p", true, "Show progress bar")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().IntVar(&maxWorkers, "workers", 4, "Maximum number of workers for parallel processing")
	rootCmd.PersistentFlags().BoolVar(&useParallel, "parallel", true, "Enable parallel processing")
	rootCmd.PersistentFlags().BoolVar(&enableFFT, "fft", true, "Enable FFT-based alignment")

	// 添加子命令
	rootCmd.AddCommand(cmd.DiffCommand())
	rootCmd.AddCommand(cmd.ApplyCommand())
	rootCmd.AddCommand(createConfigCommand())
	rootCmd.AddCommand(createBenchmarkCommand())
	rootCmd.AddCommand(createVersionCommand())

	if err := rootCmd.Execute(); err != nil {
		if logger.Sugar != nil {
			logger.Fatalf("Command execution failed: %v", err)
		} else {
			log.Fatalf("Command execution failed: %v", err)
		}
	}
}

// initializeApp 初始化应用程序
func initializeApp(cmd *cobra.Command, args []string) error {
	// 1. 加载配置
	var err error
	cfg, err = config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// 2. 命令行选项覆盖配置文件
	if cmd.Flag("repo").Changed {
		cfg.RepoDir = repoDir
	}
	if cmd.Flag("log-level").Changed {
		cfg.LogLevel = logLevel
	}
	if cmd.Flag("progress").Changed {
		cfg.ShowProgress = showProgress
	}
	if cmd.Flag("verbose").Changed {
		cfg.Verbose = verbose
	}
	if cmd.Flag("workers").Changed {
		cfg.MaxWorkers = maxWorkers
	}
	if cmd.Flag("parallel").Changed {
		cfg.UseParallel = useParallel
	}
	if cmd.Flag("fft").Changed {
		cfg.EnableFFT = enableFFT
	}

	// 3. 初始化日志系统
	loggerConfig := logger.LoggerConfig{
		Level:      cfg.LogLevel,
		OutputPath: "", // 只输出到控制台
	}

	if cfg.Verbose {
		// 在 verbose 模式下启用文件日志
		logDir := filepath.Join(cfg.RepoDir, "logs")
		loggerConfig.OutputPath = filepath.Join(logDir, "bindiff.log")
	}

	if err := logger.InitLogger(loggerConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// 4. 设置全局配置
	cmd.SetContext(cmd.Context())

	// 5. 输出启动信息
	logger.Infof("BindDiff v2.0 started with config: workers=%d, fft=%t, parallel=%t",
		cfg.MaxWorkers, cfg.EnableFFT, cfg.UseParallel)

	// 6. 创建仓库目录
	if err := os.MkdirAll(cfg.RepoDir, 0755); err != nil {
		return fmt.Errorf("failed to create repo directory: %w", err)
	}

	return nil
}

// cleanupApp 清理应用程序
func cleanupApp(cmd *cobra.Command, args []string) {
	logger.Info("BindDiff operation completed")
	logger.Close()
}

// createConfigCommand 创建配置命令
func createConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Create, view, or modify BindDiff configuration",
	}

	// 创建默认配置
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := "bindiff.yaml"
			if len(args) > 0 {
				configPath = args[0]
			}

			defaultConfig := config.DefaultConfig()
			if err := defaultConfig.SaveConfig(configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			logger.Infof("Default configuration saved to %s", configPath)
			return nil
		},
	})

	// 显示当前配置
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Current Configuration:\n")
			fmt.Printf("  Block Size: %d bytes\n", cfg.BlockSize)
			fmt.Printf("  Min Match Length: %d bytes\n", cfg.MinMatchLength)
			fmt.Printf("  Max Memory: %d MB\n", cfg.MaxMemoryMB)
			fmt.Printf("  Max Workers: %d\n", cfg.MaxWorkers)
			fmt.Printf("  Enable FFT: %t\n", cfg.EnableFFT)
			fmt.Printf("  Use Parallel: %t\n", cfg.UseParallel)
			fmt.Printf("  Show Progress: %t\n", cfg.ShowProgress)
			fmt.Printf("  Log Level: %s\n", cfg.LogLevel)
			fmt.Printf("  Repo Dir: %s\n", cfg.RepoDir)
			return nil
		},
	})

	return cmd
}

// createBenchmarkCommand 创建基准测试命令
func createBenchmarkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "benchmark [old_file] [new_file]",
		Short: "Run performance benchmark",
		Long:  "Benchmark the diff algorithm performance with different configurations",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBenchmark(args[0], args[1])
		},
	}
}

// createVersionCommand 创建版本命令
func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("BindDiff v2.0 - Enhanced Binary Diff Tool")
			fmt.Println("Features:")
			fmt.Println("  - FFT-based alignment optimization")
			fmt.Println("  - Parallel processing support")
			fmt.Println("  - Advanced hash-based matching")
			fmt.Println("  - Progress tracking and logging")
			fmt.Println("  - Configurable compression")
		},
	}
}

// runBenchmark 运行基准测试
func runBenchmark(_ string, _ string) error {
	logger.Info("Running benchmark...")
	// TODO: 实现基准测试逻辑
	return fmt.Errorf("benchmark not yet implemented")
}

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *config.Config {
	return cfg
}
