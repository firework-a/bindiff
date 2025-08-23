package cmd

import (
	"bindiff/core"
	"bindiff/pkg/config"
	"bindiff/pkg/logger"
	"bindiff/pkg/utils"
	"bindiff/types"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// DiffCommand 创建差分命令（增强版）
func DiffCommand() *cobra.Command {
	var (
		outFile      string
		showProgress bool
		useFFT       bool
		useParallel  bool
		maxWorkers   int
		blockSize    int
		minMatch     int
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "diff OLD NEW",
		Short: "Generate enhanced binary diff patch from OLD and NEW files",
		Long: `Generate an optimized binary diff patch between two files using:
- FFT-based alignment for better matching
- Parallel processing for large files
- Advanced hash-based block matching
- Configurable compression settings`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(args[0], args[1], DiffOptions{
				OutputFile:   outFile,
				ShowProgress: showProgress,
				UseFFT:       useFFT,
				UseParallel:  useParallel,
				MaxWorkers:   maxWorkers,
				BlockSize:    blockSize,
				MinMatch:     minMatch,
				Timeout:      timeout,
			})
		},
	}

	// 命令选项
	cmd.Flags().StringVarP(&outFile, "output", "o", "", "Output patch file name (default: patch.bdf)")
	cmd.Flags().BoolVar(&showProgress, "progress", true, "Show progress bar")
	cmd.Flags().BoolVar(&useFFT, "fft", true, "Enable FFT-based alignment")
	cmd.Flags().BoolVar(&useParallel, "parallel", true, "Enable parallel processing")
	cmd.Flags().IntVar(&maxWorkers, "workers", 4, "Maximum number of workers")
	cmd.Flags().IntVar(&blockSize, "block-size", 1024, "Block size for matching")
	cmd.Flags().IntVar(&minMatch, "min-match", 64, "Minimum match length")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Operation timeout (0 = no timeout)")

	return cmd
}

// DiffOptions 差分选项
type DiffOptions struct {
	OutputFile   string
	ShowProgress bool
	UseFFT       bool
	UseParallel  bool
	MaxWorkers   int
	BlockSize    int
	MinMatch     int
	Timeout      time.Duration
}

// runDiff 执行差分操作
func runDiff(oldPath, newPath string, options DiffOptions) error {
	start := time.Now()
	logger.Infof("Starting diff operation: %s -> %s", oldPath, newPath)

	// 1. 验证文件存在
	if err := validateFiles(oldPath, newPath); err != nil {
		return err
	}

	// 2. 读取文件信息
	oldInfo, err := utils.GetFileInfo(oldPath)
	if err != nil {
		return fmt.Errorf("failed to get old file info: %w", err)
	}

	newInfo, err := utils.GetFileInfo(newPath)
	if err != nil {
		return fmt.Errorf("failed to get new file info: %w", err)
	}

	logger.Infof("File sizes: old=%s, new=%s",
		utils.FormatBytes(oldInfo.Size), utils.FormatBytes(newInfo.Size))

	// 3. 读取文件数据
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old file: %w", err)
	}

	newData, err := os.ReadFile(newPath)
	if err != nil {
		return fmt.Errorf("failed to read new file: %w", err)
	}

	// 4. 创建上下文（支持超时）
	ctx := context.Background()
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 5. 配置差分选项
	diffConfig := &config.Config{
		BlockSize:      options.BlockSize,
		MinMatchLength: options.MinMatch,
		MaxWorkers:     options.MaxWorkers,
		EnableFFT:      options.UseFFT,
		UseParallel:    options.UseParallel,
		ShowProgress:   options.ShowProgress,
	}

	coreDiffOptions := &core.DiffOptions{
		Config:       diffConfig,
		ShowProgress: options.ShowProgress,
		Context:      ctx,
	}

	// 6. 计算偏移量（如果启用FFT）
	var offset int32
	if options.UseFFT {
		logger.Info("Computing FFT-based alignment...")
		offset = int32(core.ComputeOffset(oldData, newData))
		logger.Infof("Computed offset: %d", offset)
	} else {
		logger.Info("FFT alignment disabled")
	}

	// 7. 计算差分
	logger.Info("Computing binary diff...")
	patches := core.DiffWithOptions(oldData, newData, coreDiffOptions)
	logger.Infof("Generated %d patches", len(patches))

	// 8. 计算统计信息
	compressionRatio := calculateCompressionRatio(patches, int64(len(newData)))
	logger.Infof("Compression ratio: %.2f%%", compressionRatio*100)

	// 9. 创建补丁文件
	diffFile := types.DiffFile{
		MagicNumber:       types.PATCH_MAGIC,
		Version:           types.PATCH_VERSION,
		OldFileNameLength: uint32(len(filepath.Base(oldPath))),
		FileName:          []byte(filepath.Base(oldPath)),
		NewFileNameLength: uint32(len(filepath.Base(newPath))),
		NewFileName:       []byte(filepath.Base(newPath)),
		OldSize:           uint32(len(oldData)),
		NewSize:           uint32(len(newData)),
		OldHash:           oldInfo.Hash,
		NewHash:           newInfo.Hash,
		Offset:            offset,
		Diff:              patches,
	}

	// 10. 编码补丁数据
	logger.Info("Encoding patch data...")
	diffBytes := core.EncodeDiffFile(diffFile)
	diffFile.DataLength = uint32(len(diffBytes))

	// 11. 写入补丁文件
	if options.OutputFile == "" {
		options.OutputFile = "patch.bdf"
	}

	if err := utils.SafeWrite(options.OutputFile, diffBytes); err != nil {
		return fmt.Errorf("failed to write patch file: %w", err)
	}

	// 12. 输出结果统计
	duration := time.Since(start)
	patchSize := int64(len(diffBytes))

	fmt.Printf("\n✓ Patch file generated: %s\n", options.OutputFile)
	fmt.Printf("  Original size: %s\n", utils.FormatBytes(int64(len(newData))))
	fmt.Printf("  Patch size: %s\n", utils.FormatBytes(patchSize))
	fmt.Printf("  Compression: %.2f%%\n", compressionRatio*100)
	fmt.Printf("  Processing time: %s\n", utils.FormatDuration(duration))
	fmt.Printf("  Patches generated: %d\n", len(patches))

	logger.Infof("Diff operation completed in %v", duration)
	return nil
}

// validateFiles 验证文件
func validateFiles(paths ...string) error {
	for _, path := range paths {
		if stat, err := os.Stat(path); err != nil {
			return fmt.Errorf("file %s not found: %w", path, err)
		} else if stat.IsDir() {
			return fmt.Errorf("%s is a directory, not a file", path)
		}
	}
	return nil
}

// calculateCompressionRatio 计算压缩率
func calculateCompressionRatio(patches []types.Patch, originalSize int64) float64 {
	var patchSize int64
	for _, patch := range patches {
		patchSize += int64(len(patch.Data))
		patchSize += 24 // 头信息大小
	}

	if originalSize == 0 {
		return 0
	}

	return float64(patchSize) / float64(originalSize)
}
