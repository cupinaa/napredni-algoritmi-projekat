package blockmanager

type BlockManager interface {
	ReadBlock(path string, blockIndex uint64) ([]byte, error)

	WriteBlock(path string, blockIndex uint64, data []byte) error

	BlockCount(path string) (uint64, error)

	Sync(path string) error

	BlockSize() int
}
