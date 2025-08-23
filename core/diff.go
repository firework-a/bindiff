package core

import (
	"bindiff/pkg/config"
	"bindiff/pkg/logger"
	"bindiff/pkg/utils"
	"bindiff/types"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"runtime"
	"time"
)

func EqualBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NextPowerOfTwo 计算大于等于n的最小的2的幂
func NextPowerOfTwo(n int) int {
	if n == 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func EncodePatch(p []types.Patch) []byte {
	buf := new(bytes.Buffer)
	for _, entry := range p {
		buf.WriteByte(byte(entry.Op))
		binary.Write(buf, binary.LittleEndian, entry.Offset)
		binary.Write(buf, binary.LittleEndian, entry.Length)
		if entry.Op == types.OP_INSERT || entry.Op == types.OP_REPLACE {
			buf.Write(entry.Data)
		}
	}
	return buf.Bytes()
}

func DecodePatch(b []byte) ([]types.Patch, error) {
	r := bytes.NewReader(b)
	var p []types.Patch
	for r.Len() > 0 {
		opByte, err := r.ReadByte()
		if err != nil {
			return p, err
		}
		op := types.Operator(opByte)

		var offset int64
		var length int64
		if err := binary.Read(r, binary.LittleEndian, &offset); err != nil {
			return p, err
		}
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return p, err
		}

		var data []byte
		if op == types.OP_INSERT || op == types.OP_REPLACE {
			data = make([]byte, length)
			if _, err := io.ReadFull(r, data); err != nil {
				return p, err
			}
		}

		p = append(p, types.Patch{
			Op:     op,
			Offset: offset,
			Length: length,
			Data:   data,
		})
	}
	return p, nil
}

func EncodeDiffFile(df types.DiffFile) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, df.MagicNumber)
	binary.Write(buf, binary.LittleEndian, df.Version)
	binary.Write(buf, binary.LittleEndian, df.OldFileNameLength)
	buf.Write(df.FileName)
	binary.Write(buf, binary.LittleEndian, df.NewFileNameLength)
	buf.Write(df.NewFileName)
	binary.Write(buf, binary.LittleEndian, df.OldSize)
	binary.Write(buf, binary.LittleEndian, df.NewSize)
	buf.Write(df.OldHash)
	buf.Write(df.NewHash)
	binary.Write(buf, binary.LittleEndian, df.Offset)

	diffBytes := EncodePatch(df.Diff)
	binary.Write(buf, binary.LittleEndian, uint32(len(diffBytes)))
	buf.Write(diffBytes)

	return buf.Bytes()
}

func DecodeDiffFile(data []byte) (types.DiffFile, error) {
	r := bytes.NewReader(data)
	df := types.DiffFile{}
	binary.Read(r, binary.LittleEndian, &df.MagicNumber)
	binary.Read(r, binary.LittleEndian, &df.Version)
	binary.Read(r, binary.LittleEndian, &df.OldFileNameLength)
	df.FileName = make([]byte, df.OldFileNameLength)
	io.ReadFull(r, df.FileName)
	binary.Read(r, binary.LittleEndian, &df.NewFileNameLength)
	df.NewFileName = make([]byte, df.NewFileNameLength)
	io.ReadFull(r, df.NewFileName)
	binary.Read(r, binary.LittleEndian, &df.OldSize)
	binary.Read(r, binary.LittleEndian, &df.NewSize)
	df.OldHash = make([]byte, 32)
	df.NewHash = make([]byte, 32)
	io.ReadFull(r, df.OldHash)
	io.ReadFull(r, df.NewHash)
	binary.Read(r, binary.LittleEndian, &df.Offset)
	binary.Read(r, binary.LittleEndian, &df.DataLength)
	diffData := make([]byte, df.DataLength)
	io.ReadFull(r, diffData)

	patch, err := DecodePatch(diffData)
	if err != nil {
		return df, err
	}
	df.Diff = patch
	return df, nil
}

// ComputeHash 计算数据哈希
func ComputeHash(data []byte) []byte {
	return utils.ComputeHash(data)
}

// ComputeHashWithProgress 带进度的哈希计算
func ComputeHashWithProgress(data []byte, showProgress bool) []byte {
	if !showProgress || len(data) < 1024*1024 { // 小于1MB不显示进度
		return ComputeHash(data)
	}

	progress := utils.NewProgressBar(int64(len(data)), "Computing hash", true)
	defer progress.Finish()

	hasher := sha256.New()
	chunkSize := 64 * 1024 // 64KB chunks

	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		hasher.Write(data[i:end])
		progress.Set(i)
	}

	return hasher.Sum(nil)
}

// buildBlockIndex was removed as it was unused

// findBestMatch and extendMatch were removed as they were unused

// findNextMatchStart was removed as it was unused

// optimizePatches 优化补丁序列
func optimizePatches(patches []types.Patch) []types.Patch {
	if len(patches) <= 1 {
		return patches
	}

	var optimized []types.Patch
	for i, patch := range patches {
		if i == 0 {
			optimized = append(optimized, patch)
			continue
		}

		last := &optimized[len(optimized)-1]
		// 合并相邻的相同类型操作
		if canMergePatches(last, &patch) {
			mergePatches(last, &patch)
		} else {
			optimized = append(optimized, patch)
		}
	}

	return optimized
}

// canMergePatches 检查是否可以合并补丁
func canMergePatches(p1, p2 *types.Patch) bool {
	return p1.Op == p2.Op &&
		p1.Op == types.OP_INSERT &&
		p1.Offset+p1.Length == p2.Offset
}

// mergePatches 合并补丁
func mergePatches(p1, p2 *types.Patch) {
	p1.Length += p2.Length
	p1.Data = append(p1.Data, p2.Data...)
}

// parallelDiff 并发差分算法
func parallelDiff(oldData, newData []byte, options *DiffOptions) []types.Patch {
	numWorkers := options.Config.MaxWorkers
	if numWorkers <= 1 {
		return sequentialDiff(oldData, newData, options)
	}

	// 将数据分割成块
	chunkSize := len(newData) / numWorkers
	if chunkSize < options.Config.BlockSize {
		return sequentialDiff(oldData, newData, options)
	}

	// 为了简化，在这个版本中我们回退到串行处理
	// 并发处理需要更复杂的协调逻辑
	logger.Info("Parallel diff requested, using sequential for compatibility")
	return sequentialDiff(oldData, newData, options)
}

// streamingDiff 流式差分算法（用于大文件）
func streamingDiff(oldData, newData []byte, options *DiffOptions) []types.Patch {
	logger.Info("Using streaming diff algorithm for large files")

	// 分块处理大文件
	chunkSize := options.Config.MaxMemoryMB * 1024 * 1024 / 4 // 使用1/4的内存限制作为块大小
	if chunkSize <= 0 {
		chunkSize = 64 * 1024 // 默认 64KB
	}

	var patches []types.Patch

	for offset := 0; offset < len(newData); offset += chunkSize {
		end := offset + chunkSize
		if end > len(newData) {
			end = len(newData)
		}

		// 为当前块计算差分
		chunkPatches := sequentialDiff(oldData, newData[offset:end], options)

		// 调整偏移量
		for i := range chunkPatches {
			chunkPatches[i].Offset += int64(offset)
		}

		patches = append(patches, chunkPatches...)

		// 强制GC以释放内存
		if offset > 0 && offset%chunkSize == 0 {
			runtime.GC()
		}
	}

	return optimizePatches(patches)
}

// DiffOptions 差分选项
type DiffOptions struct {
	Config       *config.Config
	ShowProgress bool
	Context      context.Context
}

// DiffResult 差分结果
type DiffResult struct {
	Patches          []types.Patch
	OldSize          int64
	NewSize          int64
	CompressionRatio float64
	ProcessTime      time.Duration
	Offset           int32
}

// Diff 改进的差分算法
func Diff(oldData, newData []byte) []types.Patch {
	return DiffWithOptions(oldData, newData, &DiffOptions{
		Config:       config.DefaultConfig(),
		ShowProgress: false,
		Context:      context.Background(),
	})
}

// DiffWithOptions 使用选项的差分算法
func DiffWithOptions(oldData, newData []byte, options *DiffOptions) []types.Patch {
	start := time.Now()
	defer func() {
		logger.Infof("Diff completed in %v", time.Since(start))
	}()

	if options == nil {
		options = &DiffOptions{
			Config:       config.DefaultConfig(),
			ShowProgress: false,
			Context:      context.Background(),
		}
	}

	// 内存使用检查
	totalSize := int64(len(oldData) + len(newData))
	maxMemory := int64(options.Config.MaxMemoryMB) * 1024 * 1024
	if totalSize > maxMemory {
		logger.Warnf("Data size (%s) exceeds memory limit (%s), using streaming mode",
			utils.FormatBytes(totalSize), utils.FormatBytes(maxMemory))
		return streamingDiff(oldData, newData, options)
	}

	// 使用并发或串行处理
	if options.Config.UseParallel && len(oldData) > options.Config.BlockSize*10 {
		return parallelDiff(oldData, newData, options)
	}

	return sequentialDiff(oldData, newData, options)
}

// sequentialDiff 串行差分算法
func sequentialDiff(oldData, newData []byte, options *DiffOptions) []types.Patch {
	var patches []types.Patch
	var progress *utils.ProgressBar

	if options.ShowProgress {
		progress = utils.NewProgressBar(int64(len(newData)), "Computing diff", true)
		defer progress.Finish()
	}

	// 简化的差分算法：直接比较字节
	minLen := len(oldData)
	if len(newData) < minLen {
		minLen = len(newData)
	}

	i := 0
	for i < minLen {
		// 检查上下文取消
		select {
		case <-options.Context.Done():
			logger.Warn("Diff operation cancelled")
			return patches
		default:
		}

		// 更新进度
		if progress != nil {
			progress.Set(i)
		}

		if oldData[i] == newData[i] {
			// 相同的数据，记录 COPY 操作
			start := i
			for i < minLen && oldData[i] == newData[i] {
				i++
			}
			patches = append(patches, types.Patch{
				Op:     types.OP_COPY,
				Offset: int64(start),
				Length: int64(i - start),
			})
		} else {
			// 不同的数据，记录 REPLACE 操作
			start := i
			for i < minLen && oldData[i] != newData[i] {
				i++
			}
			patches = append(patches, types.Patch{
				Op:     types.OP_REPLACE,
				Offset: int64(start),
				Length: int64(i - start),
				Data:   newData[start:i],
			})
		}
	}

	// 处理尾部数据
	if len(newData) > minLen {
		// 新数据更长，需要 INSERT
		patches = append(patches, types.Patch{
			Op:     types.OP_INSERT,
			Offset: int64(minLen),
			Length: int64(len(newData) - minLen),
			Data:   newData[minLen:],
		})
	} else if len(oldData) > minLen {
		// 旧数据更长，需要 DELETE
		patches = append(patches, types.Patch{
			Op:     types.OP_DELETE,
			Offset: int64(minLen),
			Length: int64(len(oldData) - minLen),
		})
	}

	return patches
}

// OptimizePatches 优化补丁序列，合并相邻的操作
func OptimizePatches(patches []types.Patch) []types.Patch {
	if len(patches) <= 1 {
		return patches
	}

	var optimized []types.Patch
	for i := 0; i < len(patches); i++ {
		current := patches[i]

		// 查找相邻的可合并操作
		for i+1 < len(patches) {
			next := patches[i+1]

			// 合并相邻的INSERT操作
			if current.Op == types.OP_INSERT && next.Op == types.OP_INSERT &&
				current.Offset+current.Length == next.Offset {
				current.Length += next.Length
				current.Data = append(current.Data, next.Data...)
				i++
				continue
			}

			// 合并相邻的COPY操作
			if current.Op == types.OP_COPY && next.Op == types.OP_COPY &&
				current.Offset+current.Length == next.Offset {
				current.Length += next.Length
				i++
				continue
			}

			break
		}

		optimized = append(optimized, current)
	}

	return optimized
}

// ApplyPatch 应用补丁（改进版本）
func ApplyPatch(oldData []byte, patch []types.Patch) []byte {
	return ApplyPatchWithOptions(oldData, patch, &ApplyOptions{
		Config:       config.DefaultConfig(),
		ShowProgress: false,
		Context:      context.Background(),
	})
}

// ApplyOptions 应用补丁选项
type ApplyOptions struct {
	Config       *config.Config
	ShowProgress bool
	Context      context.Context
	VerifyResult bool
}

// ApplyPatchWithOptions 使用选项应用补丁
func ApplyPatchWithOptions(oldData []byte, patches []types.Patch, options *ApplyOptions) []byte {
	start := time.Now()
	defer func() {
		logger.Infof("Patch applied in %v", time.Since(start))
	}()

	if options == nil {
		options = &ApplyOptions{
			Config:       config.DefaultConfig(),
			ShowProgress: false,
			Context:      context.Background(),
			VerifyResult: true,
		}
	}

	// 估算结果大小
	var estimatedSize int64
	for _, p := range patches {
		switch p.Op {
		case types.OP_INSERT, types.OP_REPLACE:
			estimatedSize += p.Length
		case types.OP_COPY, types.OP_MATCH:
			estimatedSize += p.Length
		}
	}

	// 预分配结果缓冲区
	newData := make([]byte, 0, estimatedSize)
	var progress *utils.ProgressBar

	if options.ShowProgress {
		progress = utils.NewProgressBar(int64(len(patches)), "Applying patches", true)
		defer progress.Finish()
	}

	cursor := 0
	for i, patch := range patches {
		// 检查上下文取消
		select {
		case <-options.Context.Done():
			logger.Warn("Patch application cancelled")
			return newData
		default:
		}

		// 更新进度
		if progress != nil {
			progress.Set(i)
		}

		// 验证偏移量
		if int(patch.Offset) > len(oldData) {
			logger.Warnf("Patch offset %d exceeds old data length %d, skipping",
				patch.Offset, len(oldData))
			continue
		}

		// 复制中间的数据
		if int(patch.Offset) > cursor {
			newData = append(newData, oldData[cursor:patch.Offset]...)
			cursor = int(patch.Offset)
		}

		// 应用操作
		switch patch.Op {
		case types.OP_INSERT:
			newData = append(newData, patch.Data...)
		case types.OP_REPLACE:
			cursor += int(patch.Length)
			newData = append(newData, patch.Data...)
		case types.OP_DELETE:
			cursor += int(patch.Length)
		case types.OP_COPY, types.OP_MATCH:
			endPos := cursor + int(patch.Length)
			if endPos > len(oldData) {
				logger.Warnf("Copy operation exceeds old data bounds, truncating")
				endPos = len(oldData)
			}
			if cursor < len(oldData) && endPos > cursor {
				newData = append(newData, oldData[cursor:endPos]...)
				cursor = endPos
			}
		default:
			logger.Warnf("Unknown patch operation: %d", patch.Op)
		}
	}

	// 复制剩余数据
	if cursor < len(oldData) {
		newData = append(newData, oldData[cursor:]...)
	}

	return newData
}
