package service

type ArchiveMounter interface {
	// MountFile mounts a file into the storage.
	MountFile(fileName string, mountPoint string) error

	// UnmountFile unmounts a file from the storage.
	UnmountFile(mountPoint string) error
}
