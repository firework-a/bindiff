package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar 进度条管理器
type ProgressBar struct {
	bar     *progressbar.ProgressBar
	enabled bool
}

// NewProgressBar 创建进度条
func NewProgressBar(max int64, description string, enabled bool) *ProgressBar {
	if !enabled {
		return &ProgressBar{enabled: false}
	}

	bar := progressbar.NewOptions64(max,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)

	return &ProgressBar{
		bar:     bar,
		enabled: true,
	}
}

// Add 更新进度
func (p *ProgressBar) Add(num int) {
	if p.enabled && p.bar != nil {
		p.bar.Add(num)
	}
}

// Set 设置进度
func (p *ProgressBar) Set(num int) {
	if p.enabled && p.bar != nil {
		p.bar.Set(num)
	}
}

// Finish 完成进度条
func (p *ProgressBar) Finish() {
	if p.enabled && p.bar != nil {
		p.bar.Finish()
	}
}

// FileInfo 文件信息结构
type FileInfo struct {
	Path    string
	Size    int64
	Hash    []byte
	ModTime time.Time
}

// GetFileInfo 获取文件信息
func GetFileInfo(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to compute hash for %s: %w", path, err)
	}

	return &FileInfo{
		Path:    path,
		Size:    stat.Size(),
		Hash:    hasher.Sum(nil),
		ModTime: stat.ModTime(),
	}, nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// SafeWrite 安全写入文件（原子操作）
func SafeWrite(filename string, data []byte) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	// 写入临时文件
	tmpFile := filename + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// 原子性重命名
	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile) // 清理临时文件
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// BackupFile 备份文件
func BackupFile(filename string) error {
	backupName := filename + ".backup." + time.Now().Format("20060102-150405")

	src, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupName)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

// ComputeHash 计算数据的 SHA256 哈希
func ComputeHash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// CompareHashes 比较两个哈希值
func CompareHashes(hash1, hash2 []byte) bool {
	if len(hash1) != len(hash2) {
		return false
	}
	for i := range hash1 {
		if hash1[i] != hash2[i] {
			return false
		}
	}
	return true
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration 格式化持续时间
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// GetMemoryUsage 获取当前内存使用情况
func GetMemoryUsage() (float64, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024, nil // MB
}

// ErrorWithContext 带上下文的错误
type ErrorWithContext struct {
	Err     error
	Context map[string]interface{}
	Stack   []string
}

// Error 实现 error 接口
func (e *ErrorWithContext) Error() string {
	var parts []string
	parts = append(parts, e.Err.Error())

	if len(e.Context) > 0 {
		parts = append(parts, "context:")
		for k, v := range e.Context {
			parts = append(parts, fmt.Sprintf("  %s: %v", k, v))
		}
	}

	return strings.Join(parts, "\n")
}

// Unwrap 支持错误链
func (e *ErrorWithContext) Unwrap() error {
	return e.Err
}

// NewErrorWithContext 创建带上下文的错误
func NewErrorWithContext(err error, context map[string]interface{}) *ErrorWithContext {
	return &ErrorWithContext{
		Err:     err,
		Context: context,
		Stack:   getStackTrace(),
	}
}

// getStackTrace 获取调用栈
func getStackTrace() []string {
	var stack []string
	for i := 2; i < 10; i++ { // 跳过当前函数和调用者
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		stack = append(stack, fmt.Sprintf("%s:%d", filepath.Base(file), line))
	}
	return stack
}

// Retry 重试机制
func Retry(attempts int, delay time.Duration, fn func() error) error {
	var lastErr error

	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if i < attempts-1 {
			time.Sleep(delay)
			delay *= 2 // 指数退避
		}
	}

	return fmt.Errorf("failed after %d attempts, last error: %w", attempts, lastErr)
}

// TempFile 创建临时文件
func TempFile(prefix string) (*os.File, error) {
	return os.CreateTemp("", prefix+"_*.tmp")
}

// CleanupTempFiles 清理临时文件
func CleanupTempFiles(pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			// 记录但不返回错误
			fmt.Printf("Warning: failed to remove temp file %s: %v\n", match, err)
		}
	}

	return nil
}
