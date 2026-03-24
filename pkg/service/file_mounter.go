package service

// FileMounter mounts and unmounts ISO files.
type FileMounter interface {
	MountFile(fileName string, mountPoint string) error
	UnmountFile(mountPoint string) error
}
