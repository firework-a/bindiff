package cmd

import (
	"bindiff/core"
	"bindiff/pkg/logger"
	"bindiff/pkg/utils"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// ApplyCommand 创建应用补丁命令（增强版）
func ApplyCommand() *cobra.Command {
	var (
		outFile      string
		showProgress bool
		verifyResult bool
		backupOrig   bool
		timeout      time.Duration
	)

	cmd := &cobra.Command{
		Use:   "apply OLD PATCH",
		Short: "Apply a binary patch to OLD file and produce a new file",
		Long: `Apply a binary patch with enhanced safety features:
- Hash verification for input and output files
- Progress tracking for large files
- Automatic backup of original files
- Detailed error reporting and logging`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runApply(args[0], args[1], ApplyOptions{
				OutputFile:     outFile,
				ShowProgress:   showProgress,
				VerifyResult:   verifyResult,
				BackupOriginal: backupOrig,
				Timeout:        timeout,
			})
		},
	}

	// 命令选项
	cmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file name (default: from patch metadata)")
	cmd.Flags().BoolVar(&showProgress, "progress", true, "Show progress bar")
	cmd.Flags().BoolVar(&verifyResult, "verify", true, "Verify result file hash")
	cmd.Flags().BoolVar(&backupOrig, "backup", false, "Backup original file")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Operation timeout (0 = no timeout)")

	return cmd
}

// ApplyOptions 应用补丁选项
type ApplyOptions struct {
	OutputFile     string
	ShowProgress   bool
	VerifyResult   bool
	BackupOriginal bool
	Timeout        time.Duration
}

// runApply 执行补丁应用操作
func runApply(oldPath, patchPath string, options ApplyOptions) error {
	start := time.Now()
	logger.Infof("Starting apply operation: %s + %s", oldPath, patchPath)

	// 1. 验证文件存在
	if err := validateFiles(oldPath, patchPath); err != nil {
		return err
	}

	// 2. 备份原文件（如果需要）
	if options.BackupOriginal {
		logger.Info("Creating backup of original file...")
		if err := utils.BackupFile(oldPath); err != nil {
			logger.Warnf("Failed to backup original file: %v", err)
		}
	}

	// 3. 读取文件
	oldData, err := os.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read old file: %w", err)
	}

	patchBytes, err := os.ReadFile(patchPath)
	if err != nil {
		return fmt.Errorf("failed to read patch file: %w", err)
	}

	logger.Infof("File sizes: original=%s, patch=%s",
		utils.FormatBytes(int64(len(oldData))), utils.FormatBytes(int64(len(patchBytes))))

	// 4. 解码补丁文件
	logger.Info("Decoding patch file...")
	df, err := core.DecodeDiffFile(patchBytes)
	if err != nil {
		return fmt.Errorf("failed to decode patch: %w", err)
	}

	logger.Infof("Patch info: %d patches, offset=%d", len(df.Diff), df.Offset)

	// 5. 验证原文件哈希
	logger.Info("Verifying original file hash...")
	calculatedHash := core.ComputeHash(oldData)
	if !utils.CompareHashes(calculatedHash, df.OldHash) {
		return fmt.Errorf("hash mismatch: input file does not match patch source\nExpected: %x\nActual: %x",
			df.OldHash, calculatedHash)
	}

	// 6. 创建上下文（支持超时）
	ctx := context.Background()
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	}

	// 7. 应用补丁
	logger.Info("Applying patches...")
	applyOptions := &core.ApplyOptions{
		ShowProgress: options.ShowProgress,
		Context:      ctx,
		VerifyResult: options.VerifyResult,
	}

	newData := core.ApplyPatchWithOptions(oldData, df.Diff, applyOptions)

	// 8. 验证结果哈希（如果启用）
	if options.VerifyResult {
		logger.Info("Verifying result file hash...")
		resultHash := core.ComputeHash(newData)
		if !utils.CompareHashes(resultHash, df.NewHash) {
			return fmt.Errorf("result hash mismatch: patch application failed\nExpected: %x\nActual: %x",
				df.NewHash, resultHash)
		}
	}

	// 9. 确定输出文件名
	if options.OutputFile == "" {
		options.OutputFile = string(df.NewFileName)
	}

	// 10. 写入结果文件
	logger.Infof("Writing result to %s", options.OutputFile)
	if err := utils.SafeWrite(options.OutputFile, newData); err != nil {
		return fmt.Errorf("failed to write new file: %w", err)
	}

	// 11. 输出结果统计
	duration := time.Since(start)

	fmt.Printf("\n✓ Patch applied successfully: %s\n", options.OutputFile)
	fmt.Printf("  Original size: %s\n", utils.FormatBytes(int64(len(oldData))))
	fmt.Printf("  Result size: %s\n", utils.FormatBytes(int64(len(newData))))
	fmt.Printf("  Processing time: %s\n", utils.FormatDuration(duration))
	fmt.Printf("  Patches applied: %d\n", len(df.Diff))

	if options.VerifyResult {
		fmt.Printf("  ✓ Hash verification: PASSED\n")
	}

	logger.Infof("Apply operation completed in %v", duration)
	return nil
}
