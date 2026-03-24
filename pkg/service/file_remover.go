package service

// FileRemover deletes root files and queries their integrity hash.
type FileRemover interface {
	DeleteFile(fileName string) error
	GetHashFromFileInfos(fileName string) (string, error)
}
