package types


const (
	PATCH_MAGIC      = 0x42444646 // 'BDFF' magic number
	PATCH_VERSION    = 1
	INDEX_FILE       = ".binary_index"
	BLOCK_SIZE       = 1024
	MIN_MATCH_LENGTH = 64
)

type Operator uint8

// 补丁操作类型
const (
	OP_COPY    Operator = 0x01
	OP_INSERT  Operator = 0x02
	OP_REPLACE Operator = 0x03
	OP_MATCH   Operator = 0x04
	OP_DELETE  Operator = 0x05
)

// 仓库管理功能
type IndexEntry struct {
	Path      string `json:"path"`
	Size      int    `json:"size"`
	Hash      string `json:"hash"`
	Timestamp int64  `json:"timestamp"`
}

type RepositoryIndex struct {
	Version int                   `json:"version"`
	Files   map[string]IndexEntry `json:"files"`
}

type Patch struct {
	Op     Operator
	Offset int64
	Length int64
	Data   []byte
}

type Patchs struct {
	Entries []Patch
}

// +----------------------------------+
// |             Magic Number          | 4 bytes (0x42444646 = 'BDFF')
// +----------------------------------+
// |            Version Number         | 4 bytes (little-endian)
// +----------------------------------+
// |        Old File Name Length       | 4 bytes (little-endian)
// +----------------------------------+
// |         Old File Name             | Variable length
// +----------------------------------+
// |        New File Name Length       | 4 bytes (little-endian)
// +----------------------------------+
// |         New File Name             | Variable length
// +----------------------------------+
// |         Old File Size             | 4 bytes (little-endian)
// +----------------------------------+
// |         New File Size             | 4 bytes (little-endian)
// +----------------------------------+
// |         Old File SHA256           | 32 bytes
// +----------------------------------+
// |         New File SHA256           | 32 bytes
// +----------------------------------+
// |           Offset Value            | 4 bytes (signed int32, little-endian)
// +----------------------------------+
// |        Diff Data Length           | 4 bytes (little-endian)
// +----------------------------------+
// |           Diff Data               | Variable length

type DiffFile struct {
	MagicNumber uint32
	Version uint32
	OldFileNameLength uint32
	FileName []byte
	NewFileNameLength uint32
	NewFileName []byte
	OldSize uint32
	NewSize uint32
	OldHash []byte
	NewHash []byte
	Offset int32
	DataLength uint32
	Diff []Patch
}

