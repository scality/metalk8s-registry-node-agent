package library

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/moby/sys/mountinfo"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"golang.org/x/sys/unix"
)

const (
	FileSystemDefaultFileMode          = 0o600
	FileSystemDefaultDirMode           = 0o700
	FileSystemBucketPrefix             = ".bucket."
	FileSystemMultipartMetaSuffix      = ".meta"
	FileSystemMultipartPartsSuffix     = ".parts"
	FileSystemMultipartRecipientSuffix = ".recipient"
	devicesDir                         = "/devices"
	devDir                             = "/dev"

	namingRegexpValidationString = `^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{1,98}[a-zA-Z0-9]$`
)

// namingRegexp represents the naming convention for the files and buckets.
var namingRegexp = regexp.MustCompile(namingRegexpValidationString)

// enforceNamingConventions enforces the naming convention for the files and buckets.
func EnforceNamingConventions(
	objectName string,
) error {
	if !namingRegexp.MatchString(objectName) {
		return errors.Wrap(domain.ErrBusinessRuleViolation,
			errors.WithIdentifier(8),
			errors.WithDetail("object name does not match the naming regex"),
			errors.WithProperty("object_name", objectName),
			errors.WithProperty("naming_regex", namingRegexp.String()),
		)
	}

	for _, forbidden := range []string{
		"__", "--", "..",
	} {
		if strings.Contains(objectName, forbidden) {
			return errors.Wrap(domain.ErrBusinessRuleViolation,
				errors.WithIdentifier(9),
				errors.WithDetail("object name contains a forbidden particle"),
				errors.WithProperty("object_name", objectName),
				errors.WithProperty("forbidden_particle", forbidden),
			)
		}
	}

	for _, reserved := range []string{
		FileSystemBucketPrefix,
		FileSystemMultipartMetaSuffix,
		FileSystemMultipartPartsSuffix,
		FileSystemMultipartRecipientSuffix,
	} {
		if strings.Contains(objectName, reserved) {
			return errors.Wrap(domain.ErrBusinessRuleViolation,
				errors.WithIdentifier(10),
				errors.WithDetail("object name contains a reserved word"),
				errors.WithProperty("object_name", objectName),
				errors.WithProperty("reserved_word", reserved),
			)
		}
	}

	return nil
}

/*
 *
 * Filesystem Objects Checks
 *
 */

// CheckDir performs some basic checks on the directory.
// dirPath must be the absolute path to the directory.
func CheckDir(
	dirPath string,
) error {
	stat, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(11),
			errors.WithDetail("directory not found"),
			errors.WithProperty("dir_path", dirPath),
		)
	}

	if err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(12),
			errors.WithDetail("unexpected error while checking the directory"),
			errors.WithProperty("dir_path", dirPath),
			errors.CausedBy(err),
		)
	}

	if !stat.IsDir() {
		return errors.Wrap(domain.ErrConflict,
			errors.WithIdentifier(13),
			errors.WithDetail("the directory is a file"),
			errors.WithProperty("dir_path", dirPath),
		)
	}

	return nil
}

// checkFile performs some basic checks on the file.
// filePath must be the absolute path to the file.
func CheckFile(
	filePath string,
) error {
	stat, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(11),
			errors.WithDetail("file not found"),
			errors.WithProperty("file_path", filePath),
		)
	}

	if err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(12),
			errors.WithDetail("unexpected error while checking the file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	if stat.IsDir() {
		return errors.Wrap(domain.ErrConflict,
			errors.WithIdentifier(13),
			errors.WithDetail("the file is a directory"),
			errors.WithProperty("file_path", filePath),
		)
	}

	return nil
}

// saveFile saves the content to the file.
func SaveFile(
	filePath string,
	content io.Reader,
	perm os.FileMode,
) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(14),
			errors.WithDetail("unable to open the file to save the content"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	_, err = io.Copy(file, content)
	if err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(15),
			errors.WithDetail("unable to write the content to the file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	if closer, ok := content.(io.Closer); ok {
		closer.Close() //nolint:revive,errcheck // I hate you, revive
	}

	return nil
}

// createEmptyFile creates an empty file of the given size.
func CreateEmptyFile(
	filePath string,
	size int64,
	perm os.FileMode,
) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(16),
			errors.WithDetail("unable to open the file to create an empty file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	if err := file.Truncate(size); err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(17),
			errors.WithDetail("unable to truncate the file to the given size"),
			errors.WithProperty("file_path", filePath),
			errors.WithProperty("size", size),
			errors.CausedBy(err),
		)
	}

	return nil
}

// getFile returns the content of the file.
func GetFile(
	filePath string,
) (io.ReadCloser, error) {
	err := CheckFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(18),
			errors.WithDetail("check file error on getting file"),
			errors.WithProperty("file_path", filePath),
		)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(19),
			errors.WithDetail("unexpected error while opening the file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	return file, nil
}

// GetPart returns the content of the part.
func GetPart(
	filePath string,
	start int64,
	end int64,
) (io.ReadCloser, error) {
	if start < 0 || end < 0 {
		return nil, errors.Wrap(domain.ErrBadRequest,
			errors.WithIdentifier(20),
			errors.WithDetail("start or end is less than 0"),
			errors.WithProperty("file_path", filePath),
			errors.WithProperty("start", start),
			errors.WithProperty("end", end),
		)
	}
	if end < start {
		return nil, errors.Wrap(domain.ErrBadRequest,
			errors.WithIdentifier(21),
			errors.WithDetail("end is less than start"),
			errors.WithProperty("file_path", filePath),
			errors.WithProperty("end", end),
			errors.WithProperty("start", start),
		)
	}
	err := CheckFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(22),
			errors.WithDetail("check file error on getting part"),
		)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(23),
			errors.WithDetail("unexpected error while opening the file to get part"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	// Stream the byte range via SectionReader (ReaderAt) instead of buffering the full part.
	n := end - start + 1
	sr := io.NewSectionReader(file, start, n)
	return &struct {
		io.Reader
		io.Closer
	}{
		Reader: sr,
		Closer: file,
	}, nil
}

// deleteFile deletes the file.
func DeleteFile(
	filePath string,
) error {
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(24),
			errors.WithDetail("unexpected error while deleting the file"),
			errors.WithProperty("file_path", filePath),
			errors.CausedBy(err),
		)
	}

	return nil
}

// HashFile calculates the hash of the file.
func HashFile(
	filePath string,
) (string, error) {
	file, err := GetFile(filePath)
	if err != nil {
		return "", errors.Wrap(err,
			errors.WithIdentifier(25),
			errors.WithDetail("get file error on hashing file"),
		)
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	hash, err := hashReader(file)
	if err != nil {
		return "", errors.Wrap(err,
			errors.WithProperty("file_path", filePath),
		)
	}

	// Best-effort: drop the page cache populated by the full-archive read so a
	// large archive does not stay resident as reclaimable cache after hashing.
	if osFile, ok := file.(*os.File); ok {
		_ = DropCache(osFile, 0, 0) // nolint: errcheck // Best-effort cache hint.
	}

	return hash, nil
}

// hashReader calculates the hash of the reader.
func hashReader(
	reader io.Reader,
) (string, error) {
	hasher := crypto.SHA256.New()

	if _, err := io.Copy(hasher, reader); err != nil {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(26),
			errors.WithDetail("error while hashing the content"),
			errors.CausedBy(err),
		)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// DropCache advises the kernel to drop the page cache for the given byte range.
// Archive parts are written once and never re-read on the upload path,
// so retaining their pages only inflates the container's reclaimable page cache
// (charged to the cgroup's memory.usage).
// A length of 0 covers from offset to the end of the file.
func DropCache(file *os.File, offset, length int64) error {
	if err := unix.Fadvise(int(file.Fd()), offset, length, unix.FADV_DONTNEED); err != nil {
		return errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(242),
			errors.WithDetail("unable to advise the kernel to drop the page cache"),
			errors.CausedBy(err),
		)
	}

	return nil
}

// GenBucketName generates a unique bucket name for a session based on the solution archive's name and version.
func GenBucketName(solutionArchive *domain.SolutionArchive) string {
	return fmt.Sprintf("%s-%s", solutionArchive.Name, solutionArchive.Version)
}

// GenSolutionArchiveFileName generates a unique solution archive file name.
func GenSolutionArchiveFileName(solutionArchive *domain.SolutionArchive) string {
	return fmt.Sprintf("%s-%s.iso", solutionArchive.Name, solutionArchive.Version)
}

// GenSolutionDirName generates a unique solution directory name.
func GenSolutionDirName(solutionArchive *domain.SolutionArchive) string {
	return fmt.Sprintf("%s/%s", solutionArchive.Name, solutionArchive.Version)
}

// solutionArchiveExists returns if the solution archive is already present in
// the final storage.
func SolutionArchiveExists(
	solutionArchive *domain.SolutionArchive,
	fileNames []string,
) bool {
	fileName := GenSolutionArchiveFileName(solutionArchive)
	return slices.Contains(fileNames, fileName)
}

// ExtractSessionBucket validate and extract the session bucket from the given list of buckets.
func ExtractSessionBucket(buckets []string, solutionArchive *domain.SolutionArchive) (string, error) {
	var sessionBuckets []string

	// Only keep the session buckets
	bucketName := GenBucketName(solutionArchive)
	for _, bucket := range buckets {
		if strings.HasPrefix(bucket, bucketName) {
			sessionBuckets = append(sessionBuckets, bucket)
		}
	}

	// Check if we have the right number of session buckets
	if len(sessionBuckets) < 1 {
		return "", errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(27),
			errors.WithDetail("no session bucket found"),
		)
	} else if len(sessionBuckets) > 1 {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(28),
			errors.WithDetail("multiple session buckets found"),
		)
	}

	return sessionBuckets[0], nil
}

// GetSolutionArchiveNameVersion generates a solution archive name and version.
func GetSolutionArchiveNameVersion(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}

/*
 *
 * Miscellaneous
 *
 */

// compareSolutionArchiveMetas compares the given Solution Archive Meta instances and returns
// an error if they are different.
func CompareSolutionArchiveMetas(
	a, b *domain.SolutionArchive,
) error {
	if a == b {
		return nil
	}

	problems := make(map[string]any)

	if a.Name != b.Name {
		problems["problem_component_mismatch"] = fmt.Sprintf("%s != %s", a.Name, b.Name)
	}

	if a.Version != b.Version {
		problems["problem_version_mismatch"] = fmt.Sprintf("%s != %s", a.Version, b.Version)
	}

	// When the controller (re)starts, it tests if information in SolutionArchive CR correspond to the stored one
	// If a part has already been uploaded, the stored information will have a size.
	// As the size is not included into the CR, the controller consider a size equal to 0 as a default value.
	// So, the comparison failed need to test a.Size != 0 to allow this difference
	if a.Size != b.Size && a.Size != 0 {
		problems["problem_size_mismatch"] = fmt.Sprintf("%d != %d", a.Size, b.Size)
	}

	if (a.Hash != nil && b.Hash == nil) || (a.Hash == nil && b.Hash != nil) {
		problems["problem_hash_mismatch"] = "One hash is nil, the other is not"
	}

	if a.Hash != nil && b.Hash != nil && *a.Hash != *b.Hash {
		problems["problem_hash_mismatch"] = fmt.Sprintf("%s != %s", *a.Hash, *b.Hash)
	}

	if len(problems) > 0 {
		return errors.Wrap(domain.ErrConflict,
			errors.WithIdentifier(29),
			errors.WithDetail("solution archive metas do not match"),
			errors.WithProperty("problems", problems),
		)
	}

	return nil
}

// IsMounted checks if the given source file is mounted at the given mount point.
// If the file is not mounted, it returns false and no error.
// It returns true if correctly mounted, either it returns false and an error
func IsMounted(filePath string, mountPath string) (bool, error) {
	// Step 1: Find the loop device from the mount point
	loopDevice, err := GetLoopDeviceForMount(mountPath)
	if err != nil {
		// No mount point is present, check if the directory is empty
		if errors.Is(err, domain.ErrNotFound) {
			entries, err := os.ReadDir(mountPath)
			if err != nil {
				return false, errors.Wrap(domain.ErrMountSolutionArchiveInternal,
					errors.WithIdentifier(30),
					errors.WithDetail("unexpected error while checking the content of the directory"),
					errors.WithProperty("mount_path", mountPath),
					errors.CausedBy(err),
				)
			}
			if len(entries) > 0 {
				return false, errors.Wrap(domain.ErrMountSolutionArchiveNotEmptyDir,
					errors.WithIdentifier(31),
					errors.WithDetail("mount point is not empty"),
					errors.WithProperty("mount_path", mountPath),
				)
			}
			return false, nil
		}
		return false, errors.Wrap(err,
			errors.WithIdentifier(32),
		)
	}

	// Step 2: Find the backing file from the loop device
	backingFile, err := GetBackingFile(loopDevice)
	if err != nil {
		return false, errors.Wrap(err,
			errors.WithIdentifier(33),
			errors.WithDetail("get backing file error on checking if the file is mounted"),
			errors.WithProperty("mount_path", mountPath),
			errors.WithProperty("loop_device", loopDevice),
		)
	}

	if backingFile == filePath {
		return true, nil
	}

	return false, errors.Wrap(domain.ErrMountSolutionArchiveIncorrectMount,
		errors.WithIdentifier(34),
		errors.WithDetail("incorrect mount point"),
		errors.WithProperty("expected_mount_path", mountPath),
		errors.WithProperty("actual_backing_file_path", backingFile),
		errors.WithProperty("file_path", filePath),
	)
}

// MountISO mounts the ISO file to the given mount point.
func MountISO(isoPath, mountPoint string) error {
	// 1. Get free loop device
	ctrl, err := os.Open(fmt.Sprintf("%s/loop-control", devDir))
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(35),
			errors.WithDetail("error while opening the loop-control file"),
			errors.WithProperty("mount_point", mountPoint),
			errors.CausedBy(err),
		)
	}
	defer ctrl.Close() // nolint: errcheck // No error check on defer.

	loopNum, _, errno := unix.Syscall(unix.SYS_IOCTL, ctrl.Fd(), 0x4C82, 0) // LOOP_CTL_GET_FREE
	if errno != 0 {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(36),
			errors.WithDetail("error while calling ioctl on the loop-control file to get the free loop device"),
			errors.WithProperty("mount_point", mountPoint),
			errors.WithProperty("errno", errno),
			errors.WithProperty("ioctl_name", "LOOP_CTL_GET_FREE"),
		)
	}
	loopPath := fmt.Sprintf("%s/loop%d", devDir, loopNum)

	// 2. Attach ISO with loop device
	loop, err := os.OpenFile(loopPath, os.O_RDWR, 0)
	if err != nil {
		// ErrNotExist means targeted loop device does not exist
		if errors.Is(err, os.ErrNotExist) {
			// loop device creation in a specific directory to face permission issues on /dev
			loopPath = fmt.Sprintf("%s/loop%d", devicesDir, loopNum)
			loop, err = os.OpenFile(loopPath, os.O_RDWR, 0)
			if err != nil {
				// ErrNotExist means targeted loop device does not exist
				if errors.Is(err, os.ErrNotExist) {
					// Create loop device with major=7, minor=loopNum
					devNum := unix.Mkdev(7, uint32(loopNum))
					err = unix.Mknod(loopPath, unix.S_IFBLK|0660, int(devNum))
					if err != nil {
						return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
							errors.WithIdentifier(37),
							errors.WithDetail("error while creating loop device"),
							errors.WithProperty("loop_path", loopPath),
							errors.WithProperty("mount_point", mountPoint),
							errors.CausedBy(err),
						)
					}
					// Open the newly created loop device (mknod only creates the node).
					loop, err = os.OpenFile(loopPath, os.O_RDWR, 0)
					if err != nil {
						return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
							errors.WithIdentifier(38),
							errors.WithDetail("error while opening the loop device"),
							errors.WithProperty("loop_path", loopPath),
							errors.WithProperty("mount_point", mountPoint),
							errors.CausedBy(err),
						)
					}
				} else {
					return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
						errors.WithIdentifier(39),
						errors.WithDetail("error while opening the loop device"),
						errors.WithProperty("loop_path", loopPath),
						errors.WithProperty("mount_point", mountPoint),
						errors.CausedBy(err),
					)
				}
			}
		} else {
			return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
				errors.WithIdentifier(40),
				errors.WithDetail("error while opening the loop device"),
				errors.WithProperty("loop_path", loopPath),
				errors.WithProperty("mount_point", mountPoint),
				errors.CausedBy(err),
			)
		}
	}
	defer loop.Close() // nolint: errcheck // No error check on defer.

	iso, err := os.Open(isoPath)
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(41),
			errors.WithDetail("error while opening the ISO file"),
			errors.WithProperty("iso_path", isoPath),
			errors.CausedBy(err),
		)
	}
	defer iso.Close() // nolint: errcheck // No error check on defer.

	_, _, errno = unix.Syscall(unix.SYS_IOCTL, loop.Fd(), 0x4C00, iso.Fd()) // LOOP_SET_FD
	if errno != 0 {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(42),
			errors.WithDetail("error while calling ioctl on the loop device to set the ISO file"),
			errors.WithProperty("errno", errno),
			errors.WithProperty("iso_path", isoPath),
			errors.WithProperty("mount_point", mountPoint),
			errors.WithProperty("ioctl_name", "LOOP_SET_FD"),
		)
	}

	// 3. Mount
	err = os.MkdirAll(mountPoint, 0755)
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(43),
			errors.WithDetail("error while creating the mount point"),
			errors.WithProperty("mount_point", mountPoint),
			errors.CausedBy(err),
		)
	}
	err = unix.Mount(loopPath, mountPoint, "iso9660", unix.MS_RDONLY, "")
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(44),
			errors.WithDetail("error while mounting the ISO file"),
			errors.WithProperty("loop_device_path", loopPath),
			errors.WithProperty("iso_path", isoPath),
			errors.WithProperty("mount_point", mountPoint),
			errors.CausedBy(err),
		)
	}
	return nil
}

// UnmountISO unmounts the ISO file from the given mount point.
func UnmountISO(mountPoint string) error {
	// Retrieve loop device from a mountPoint
	loopDevice, err := GetLoopDeviceForMount(mountPoint)
	if err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(45),
			errors.WithDetail("error while getting the loop device associated to the mount point"),
			errors.WithProperty("mount_point", mountPoint),
		)
	}

	// Unmount the ISO
	err = unix.Unmount(mountPoint, 0)
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(46),
			errors.WithDetail("error while unmounting the ISO file"),
			errors.WithProperty("mount_point", mountPoint),
			errors.CausedBy(err),
		)
	}

	// Sometimes, even if ISO is unmounted, the loop device remains attached.
	// We need to clear the loop device file descriptor to avoid saturation.
	f, err := os.Open(loopDevice)
	if err != nil {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(47),
			errors.WithDetail("error while opening the loop device"),
			errors.WithProperty("loop_device", loopDevice),
			errors.WithProperty("mount_point", mountPoint),
			errors.CausedBy(err),
		)
	}
	defer f.Close() // nolint: errcheck // No error check on defer.

	_, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), 0x4C01, 0) // LOOP_CLR_FD
	// unix.ENXIO means targeted loop device is already detached
	if errno != 0 && errno != unix.ENXIO {
		return errors.Wrap(domain.ErrMountSolutionArchiveInternal,
			errors.WithIdentifier(48),
			errors.WithDetail("error while calling ioctl on the loop device to clear the file descriptor"),
			errors.WithProperty("errno", errno),
			errors.WithProperty("ioctl_name", "LOOP_CLR_FD"),
		)
	}
	return nil
}

// GetBackingFile queries a loop device (e.g., "/dev/loop0")
// and returns the path to its associated backing file (e.g., "/tmp/myimage.img").
func GetBackingFile(loopDevice string) (string, error) {
	if !strings.HasPrefix(loopDevice, fmt.Sprintf("%s/loop", devDir)) &&
		!strings.HasPrefix(loopDevice, fmt.Sprintf("%s/loop", devicesDir)) {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(49),
			errors.WithDetail("input is not a loop device"),
			errors.WithProperty("loop_device", loopDevice),
		)
	}

	// Extract the loop device number (e.g., "loop0" from "/dev/loop0")
	loopName, ok := strings.CutPrefix(loopDevice, fmt.Sprintf("%s/", devDir))
	if !ok {
		loopName = strings.TrimPrefix(loopDevice, fmt.Sprintf("%s/", devicesDir))
	}

	// Use sysfs interface to read the backing file
	// This approach doesn't require special permissions unlike opening /dev/loopX directly
	sysfsPath := "/sys/block/" + loopName + "/loop/backing_file"

	backingFileBytes, err := os.ReadFile(sysfsPath)
	if err != nil {
		// If sysfs method fails, fall back to ioctl method
		// This preserves backward compatibility and works when sysfs is not available
		backingFileIoctl, err := getBackingFileViaIoctl(loopDevice)
		if err != nil {
			return "", errors.Wrap(err,
				errors.WithIdentifier(50),
				errors.WithDetail("error while getting the backing file via ioctl"),
			)
		}
		return backingFileIoctl, nil
	}

	// The sysfs file contains the backing file path with a newline at the end
	backingFile := strings.TrimSpace(string(backingFileBytes))
	if backingFile == "" {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(51),
			errors.WithDetail("loop device is not associated with a file"),
			errors.WithProperty("loop_device", loopDevice),
		)
	}

	return backingFile, nil
}

// getBackingFileViaIoctl is a fallback method that uses ioctl to query loop device information.
// This method requires read permissions on the loop device file.
func getBackingFileViaIoctl(loopDevice string) (string, error) {
	// Open the loop device file
	file, err := os.Open(loopDevice)
	if err != nil {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(52),
			errors.WithDetail("failed to open loop device"),
			errors.WithProperty("loop_device", loopDevice),
			errors.CausedBy(err),
		)
	}
	defer file.Close() // nolint: errcheck // No error check on defer.

	// Get the file descriptor
	fd := file.Fd()

	// Prepare the struct to hold the loop info
	var info *unix.LoopInfo64

	// Call the ioctl to get the loop device status
	// This is the Go-native way to ask the kernel "what file is backing this device?"
	info, err = unix.IoctlLoopGetStatus64(int(fd))
	if err != nil {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(53),
			errors.WithDetail("ioctl LOOP_GET_STATUS64 failed"),
			errors.WithProperty("loop_device", loopDevice),
			errors.CausedBy(err),
		)
	}

	// The file name is in info.File_name, which is a fixed-size byte array.
	// We must convert it to a Go string, stopping at the first null byte.
	// unix.ByteSliceToString handles this C-style string conversion perfectly.
	backingFile := unix.ByteSliceToString(info.File_name[:])
	if backingFile == "" {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(54),
			errors.WithDetail("loop device is not associated with a file"),
			errors.WithProperty("loop_device", loopDevice),
		)
	}

	return backingFile, nil
}

// GetLoopDeviceForMount finds the loop device for a given mount point.
func GetLoopDeviceForMount(mountPath string) (string, error) {
	filter := func(info *mountinfo.Info) (skip, stop bool) {
		result := info.Mountpoint == mountPath
		return !result, result
	}

	mounts, err := mountinfo.GetMounts(filter)
	if err != nil {
		return "", errors.Wrap(domain.ErrInternal,
			errors.WithIdentifier(55),
			errors.WithDetail("unexpected error while retrieving the mounts"),
			errors.WithProperty("mount_path", mountPath),
			errors.CausedBy(err),
		)
	}

	if len(mounts) == 0 {
		return "", errors.Wrap(domain.ErrNotFound,
			errors.WithIdentifier(56),
			errors.WithDetail("no mount found"),
			errors.WithProperty("mount_path", mountPath),
		)
	}

	sourceDevice := mounts[0].Source
	if strings.HasPrefix(sourceDevice, fmt.Sprintf("%s/loop", devDir)) ||
		strings.HasPrefix(sourceDevice, fmt.Sprintf("%s/loop", devicesDir)) {
		return sourceDevice, nil
	}
	return "", errors.Wrap(domain.ErrInternal,
		errors.WithIdentifier(57),
		errors.WithDetail("source device is not a loop device"),
		errors.WithProperty("mount_path", mountPath),
		errors.WithProperty("source_device_path", sourceDevice),
	)
}

func GenBucketPath(solutionArchivesLocation string, bucketName string) string {
	return filepath.Join(solutionArchivesLocation, FileSystemBucketPrefix+bucketName)
}

// GenBaseMultipartFilePath generates the base multipart file path for the
// given bucket name and file name without apply any suffix.
func GenBaseMultipartFilePath(solutionArchivesLocation string, bucketName, fileName string) string {
	return filepath.Join(GenBucketPath(solutionArchivesLocation, bucketName), fileName)
}

// GenMultipartMetaFilePath generates the multipart meta file path for the given
// bucket name and file name.
func GenMultipartMetaFilePath(solutionArchivesLocation string, bucketName, fileName string) string {
	return GenBaseMultipartFilePath(solutionArchivesLocation, bucketName, fileName) + FileSystemMultipartMetaSuffix
}

// GenMultipartPartsFilePath generates the multipart parts file path for the
// given bucket name and file name.
func GenMultipartPartsFilePath(solutionArchivesLocation string, bucketName, fileName string) string {
	return GenBaseMultipartFilePath(solutionArchivesLocation, bucketName, fileName) + FileSystemMultipartPartsSuffix
}

// GenMultipartRecipientFilePath generates the multipart recipient file path for
// the given bucket name and file name.
func GenMultipartRecipientFilePath(solutionArchivesLocation string, bucketName, fileName string) string {
	return GenBaseMultipartFilePath(solutionArchivesLocation, bucketName, fileName) +
		FileSystemMultipartRecipientSuffix
}

// GenMultipartFilePaths generates the multipart file paths for the given bucket
// name and file name.
func GenMultipartFilePaths(
	solutionArchivesLocation string,
	bucketName,
	fileName string,
) (metaFilePath, partsFilePath, recipientFilePath string) {
	return GenMultipartMetaFilePath(solutionArchivesLocation, bucketName, fileName),
		GenMultipartPartsFilePath(solutionArchivesLocation, bucketName, fileName),
		GenMultipartRecipientFilePath(solutionArchivesLocation, bucketName, fileName)
}
