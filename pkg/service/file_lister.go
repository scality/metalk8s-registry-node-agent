package service

// FileLister enumerates root files and queries cached file metadata.
type FileLister interface {
	ListFiles() ([]string, error)
	GetSizeFromFileInfos(fileName string) (int64, error)
}
