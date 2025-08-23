package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	// 核心配置
	BlockSize      int `mapstructure:"block_size"`
	MinMatchLength int `mapstructure:"min_match_length"`
	MaxMemoryMB    int `mapstructure:"max_memory_mb"`

	// 性能配置
	MaxWorkers  int  `mapstructure:"max_workers"`
	EnableFFT   bool `mapstructure:"enable_fft"`
	UseParallel bool `mapstructure:"use_parallel"`

	// 输出配置
	ShowProgress bool   `mapstructure:"show_progress"`
	Verbose      bool   `mapstructure:"verbose"`
	LogLevel     string `mapstructure:"log_level"`

	// 文件配置
	RepoDir        string `mapstructure:"repo_dir"`
	TempDir        string `mapstructure:"temp_dir"`
	BackupOriginal bool   `mapstructure:"backup_original"`

	// 安全配置
	VerifyChecksums  bool `mapstructure:"verify_checksums"`
	CompressionLevel int  `mapstructure:"compression_level"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BlockSize:        1024,
		MinMatchLength:   64,
		MaxMemoryMB:      512,
		MaxWorkers:       4,
		EnableFFT:        true,
		UseParallel:      true,
		ShowProgress:     true,
		Verbose:          false,
		LogLevel:         "info",
		RepoDir:          ".bindiff",
		TempDir:          os.TempDir(),
		BackupOriginal:   false,
		VerifyChecksums:  true,
		CompressionLevel: 6,
	}
}

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	viper.SetDefault("block_size", config.BlockSize)
	viper.SetDefault("min_match_length", config.MinMatchLength)
	viper.SetDefault("max_memory_mb", config.MaxMemoryMB)
	viper.SetDefault("max_workers", config.MaxWorkers)
	viper.SetDefault("enable_fft", config.EnableFFT)
	viper.SetDefault("use_parallel", config.UseParallel)
	viper.SetDefault("show_progress", config.ShowProgress)
	viper.SetDefault("verbose", config.Verbose)
	viper.SetDefault("log_level", config.LogLevel)
	viper.SetDefault("repo_dir", config.RepoDir)
	viper.SetDefault("temp_dir", config.TempDir)
	viper.SetDefault("backup_original", config.BackupOriginal)
	viper.SetDefault("verify_checksums", config.VerifyChecksums)
	viper.SetDefault("compression_level", config.CompressionLevel)

	// 设置配置文件路径
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// 查找配置文件
		viper.SetConfigName("bindiff")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.bindiff")
		viper.AddConfigPath("/etc/bindiff")
	}

	// 环境变量支持
	viper.SetEnvPrefix("BINDIFF")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 解析配置
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// Validate 验证配置参数
func (c *Config) Validate() error {
	if c.BlockSize <= 0 || c.BlockSize > 1024*1024 {
		return fmt.Errorf("block_size must be between 1 and 1048576, got %d", c.BlockSize)
	}

	if c.MinMatchLength <= 0 || c.MinMatchLength > c.BlockSize {
		return fmt.Errorf("min_match_length must be between 1 and block_size(%d), got %d",
			c.BlockSize, c.MinMatchLength)
	}

	if c.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive, got %d", c.MaxMemoryMB)
	}

	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive, got %d", c.MaxWorkers)
	}

	if c.CompressionLevel < 0 || c.CompressionLevel > 9 {
		return fmt.Errorf("compression_level must be between 0 and 9, got %d", c.CompressionLevel)
	}

	// 验证日志级别
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	return nil
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig(configPath string) error {
	viper.Set("block_size", c.BlockSize)
	viper.Set("min_match_length", c.MinMatchLength)
	viper.Set("max_memory_mb", c.MaxMemoryMB)
	viper.Set("max_workers", c.MaxWorkers)
	viper.Set("enable_fft", c.EnableFFT)
	viper.Set("use_parallel", c.UseParallel)
	viper.Set("show_progress", c.ShowProgress)
	viper.Set("verbose", c.Verbose)
	viper.Set("log_level", c.LogLevel)
	viper.Set("repo_dir", c.RepoDir)
	viper.Set("temp_dir", c.TempDir)
	viper.Set("backup_original", c.BackupOriginal)
	viper.Set("verify_checksums", c.VerifyChecksums)
	viper.Set("compression_level", c.CompressionLevel)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return viper.WriteConfigAs(configPath)
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 检查环境变量
	if configPath := os.Getenv("BINDIFF_CONFIG"); configPath != "" {
		return configPath
	}

	// 检查用户目录
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".bindiff", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}
	}

	// 检查当前目录
	if _, err := os.Stat("bindiff.yaml"); err == nil {
		return "bindiff.yaml"
	}

	return ""
}
