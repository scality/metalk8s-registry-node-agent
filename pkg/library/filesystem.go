package library

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
)

const (
	FileSystemDefaultFileMode          = 0o600
	FileSystemDefaultDirMode           = 0o700
	FileSystemBucketPrefix             = ".bucket."
	FileSystemMultipartMetaSuffix      = ".meta"
	FileSystemMultipartPartsSuffix     = ".parts"
	FileSystemMultipartRecipientSuffix = ".recipient"

	namingRegexpValidationString = `^[a-zA-Z0-9][a-zA-Z0-9_\-\.]{1,98}[a-zA-Z0-9]$`
)

// namingRegexp represents the naming convention for the files and buckets.
var namingRegexp = regexp.MustCompile(namingRegexpValidationString)

// enforceNamingConventions enforces the naming convention for the files and buckets.
func EnforceNamingConventions(
	objectName string,
) error {
	if !namingRegexp.MatchString(objectName) {
		return errors.From(domain.ErrBusinessRuleViolation).
			WithIdentifier(422001).
			WithDetail("object name does not match the naming regex").
			WithProperty("object_name", objectName).
			WithProperty("naming_regex", namingRegexp.String()).
			Throw()
	}

	for _, forbidden := range []string{
		"__", "--", "..",
	} {
		if strings.Contains(objectName, forbidden) {
			return errors.From(domain.ErrBusinessRuleViolation).
				WithIdentifier(422001).
				WithDetail("object name contains a forbidden particle").
				WithProperty("object_name", objectName).
				WithProperty("forbidden_particle", forbidden).
				Throw()
		}
	}

	for _, reserved := range []string{
		FileSystemBucketPrefix,
		FileSystemMultipartMetaSuffix,
		FileSystemMultipartPartsSuffix,
		FileSystemMultipartRecipientSuffix,
	} {
		if strings.Contains(objectName, reserved) {
			return errors.From(domain.ErrBusinessRuleViolation).
				WithIdentifier(422001).
				WithDetail("object name contains a reserved word").
				WithProperty("object_name", objectName).
				WithProperty("reserved_word", reserved).
				Throw()
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
		return errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("directory not found").
			WithProperty("dir_path", dirPath).
			Throw()
	}

	if err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while checking the directory").
			WithProperty("dir_path", dirPath).
			CausedBy(err).
			Throw()
	}

	if !stat.IsDir() {
		return errors.From(domain.ErrConflict).
			WithIdentifier(409000).
			WithDetail("the directory is a file").
			WithProperty("dir_path", dirPath).
			Throw()
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
		return errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("file system not found").
			WithProperty("file_path", filePath).
			Throw()
	}

	if err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while checking the file").
			WithProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	if stat.IsDir() {
		return errors.From(domain.ErrConflict).
			WithIdentifier(409000).
			WithDetail("the file is a directory").
			WithProperty("file_path", filePath).
			Throw()
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
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unable to save file").
			WithProperty("file_path", filePath).
			WithProperty("while", "opening the file").
			CausedBy(err).
			Throw()
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	_, err = io.Copy(file, content)
	if err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unable to save file").
			WithProperty("file_path", filePath).
			WithProperty("while", "writing the file").
			CausedBy(err).
			Throw()
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
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unable to save file").
			WithProperty("file_path", filePath).
			WithProperty("while", "opening the file").
			CausedBy(err).
			Throw()
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	if err := file.Truncate(size); err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unable to truncate file").
			WithProperty("file_path", filePath).
			WithProperty("size", size).
			WithProperty("while", "truncating the file").
			CausedBy(err).
			Throw()
	}

	return nil
}

// getFile returns the content of the file.
func GetFile(
	filePath string,
) (io.ReadCloser, error) {
	err := CheckFile(filePath)
	if err != nil {
		return nil, errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("file not found").
			WithProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while opening the file").
			WithProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	return file, nil
}

// deleteFile deletes the file.
func DeleteFile(
	filePath string,
) error {
	err := os.Remove(filePath)
	if err != nil {
		return errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while deleting the file").
			WithProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	return nil
}

// hashFile calculates the hash of the file.
func HashFile(
	filePath string,
) (string, error) {
	file, err := GetFile(filePath)
	if err != nil {
		return "", errors.Stamp(err)
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	return HashReader(file)
}

// hashReader calculates the hash of the reader.
func HashReader(
	reader io.Reader,
) (string, error) {
	hasher := crypto.SHA256.New()

	if _, err := io.Copy(hasher, reader); err != nil {
		return "", errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("unexpected error while hashing the reader").
			CausedBy(err).
			Throw()
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// GenBucketName generates a unique bucket name for a session based on the solution archive's name and version.
func GenBucketName(solutionArchive *domain.SolutionArchive) string {
	return fmt.Sprintf("%s-%s", solutionArchive.Name, solutionArchive.Version)
}

// GenSolutionArchiveFileName generates a unique solution archive file name.
func GenSolutionArchiveFileName(solutionArchive *domain.SolutionArchive) string {
	return fmt.Sprintf("%s-%s.iso", solutionArchive.Name, solutionArchive.Version)
}

// solutionArchiveExists returns if the solution archive is already present in
// the final storage.
func SolutionArchiveExists(
	solutionArchive *domain.SolutionArchive,
	fileNames []string,
) bool {
	filesMap := make(map[string]any, len(fileNames))
	for _, fileName := range fileNames {
		filesMap[fileName] = nil
	}

	fileName := GenSolutionArchiveFileName(solutionArchive)
	if _, ok := filesMap[fileName]; ok {
		return true
	}
	return false
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
		return "", errors.From(domain.ErrNotFound).
			WithIdentifier(404000).
			WithDetail("no session bucket found").
			Throw()
	} else if len(sessionBuckets) > 1 {
		return "", errors.From(domain.ErrInternal).
			WithIdentifier(500000).
			WithDetail("multiple session buckets found").
			Throw()
	}

	return sessionBuckets[0], nil
}

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

	if a.Size != b.Size {
		problems["problem_size_mismatch"] = fmt.Sprintf("%d != %d", a.Size, b.Size)
	}

	if a.Hash != b.Hash {
		problems["problem_hash_mismatch"] = fmt.Sprintf("%s != %s", a.Hash, b.Hash)
	}

	if len(problems) > 0 {
		return errors.From(domain.ErrConflict).
			WithIdentifier(409000).
			WithDetail("solution archive metas do not match").
			WithProperty("problems", problems).
			Throw()
	}

	return nil
}
