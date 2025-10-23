package library

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

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
		return domain.FromTemplate(domain.ErrBusinessRuleViolationError).
			WithDetail("Object name does not match the naming regex.").
			AddProperty("object_name", objectName).
			AddProperty("naming_regex", namingRegexp.String()).
			Throw()
	}

	for _, forbidden := range []string{
		"__", "--", "..",
	} {
		if strings.Contains(objectName, forbidden) {
			return domain.FromTemplate(domain.ErrBusinessRuleViolationError).
				WithDetail("Object name contains a forbidden particle.").
				AddProperty("object_name", objectName).
				AddProperty("forbidden_particle", forbidden).
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
			return domain.FromTemplate(domain.ErrBusinessRuleViolationError).
				WithDetail("Object name contains a reserved word.").
				AddProperty("object_name", objectName).
				AddProperty("reserved_word", reserved).
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
		return domain.FromTemplate(domain.ErrNotFoundError).
			WithDetail("Directory not found.").
			AddProperty("dir_path", dirPath).
			Throw()
	}

	if err != nil {
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unexpected error while checking the directory.").
			AddProperty("dir_path", dirPath).
			CausedBy(err).
			Throw()
	}

	if !stat.IsDir() {
		return domain.FromTemplate(domain.ErrConflictError).
			WithDetail("The directory is a file").
			AddProperty("dir_path", dirPath).
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
		return domain.FromTemplate(domain.ErrNotFoundError).
			WithDetail("FileSystem not found.").
			AddProperty("file_path", filePath).
			Throw()
	}

	if err != nil {
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unexpected error while checking the file.").
			AddProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	if stat.IsDir() {
		return domain.FromTemplate(domain.ErrConflictError).
			WithDetail("The file is a directory.").
			AddProperty("file_path", filePath).
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
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unable to save file.").
			AddProperty("file_path", filePath).
			AddProperty("while", "opening the file").
			CausedBy(err).
			Throw()
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	_, err = io.Copy(file, content)
	if err != nil {
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unable to save file.").
			AddProperty("file_path", filePath).
			AddProperty("while", "writing the file").
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
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unable to save file.").
			AddProperty("file_path", filePath).
			AddProperty("while", "opening the file").
			CausedBy(err).
			Throw()
	}

	defer file.Close() // nolint: errcheck // No error check on defer.

	if err := file.Truncate(size); err != nil {
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unable to truncate file.").
			AddProperty("file_path", filePath).
			AddProperty("size", size).
			AddProperty("while", "truncating the file").
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
		return nil, domain.FromTemplate(domain.ErrNotFoundError).
			WithDetail("File not found.").
			AddProperty("file_path", filePath).
			CausedBy(err).
			Throw()
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unexpected error while opening the file.").
			AddProperty("file_path", filePath).
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
		return domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unexpected error while deleting the file.").
			AddProperty("file_path", filePath).
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
		return "", domain.Stamp(err)
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
		return "", domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Unexpected error while hashing the reader.").
			CausedBy(err).
			Throw()
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// GenBucketName generates a unique bucket name for a session based on the artifact's name and version.
func GenBucketName(artifact *domain.Artifact) string {
	return fmt.Sprintf("%s-%s", artifact.Name, artifact.Version)
}

// GenArtifactFileName generates a unique artifact file name.
func GenArtifactFileName(artifact *domain.Artifact) string {
	return fmt.Sprintf("%s-%s.iso", artifact.Name, artifact.Version)
}

// artifactExists returns if the artifact is already present in
// the final storage.
func ArtifactExists(
	artifact *domain.Artifact,
	fileNames []string,
) bool {
	filesMap := make(map[string]any, len(fileNames))
	for _, fileName := range fileNames {
		filesMap[fileName] = nil
	}

	fileName := GenArtifactFileName(artifact)
	if _, ok := filesMap[fileName]; ok {
		return true
	}
	return false
}

// ExtractSessionBucket validate and extract the session bucket from the given list of buckets.
func ExtractSessionBucket(buckets []string, artifact *domain.Artifact) (string, error) {
	var sessionBuckets []string

	// Only keep the session buckets
	bucketName := GenBucketName(artifact)
	for _, bucket := range buckets {
		if strings.HasPrefix(bucket, bucketName) {
			sessionBuckets = append(sessionBuckets, bucket)
		}
	}

	// Check if we have the right number of session buckets
	if len(sessionBuckets) < 1 {
		return "", domain.FromTemplate(domain.ErrNotFoundError).
			WithDetail("No session bucket found.").
			Throw()
	} else if len(sessionBuckets) > 1 {
		return "", domain.FromTemplate(domain.ErrInternalError).
			WithDetail("Multiple session buckets found.").
			Throw()
	}

	return sessionBuckets[0], nil
}

/*
 *
 * Miscellaneous
 *
 */

// compareArtifactMetas compares the given ArtifactMeta instances and returns
// an error if they are different.
func CompareArtifactMetas(
	a, b *domain.Artifact,
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
		return domain.FromTemplate(domain.ErrConflictError).
			WithDetail("Artifact metas do not match.").
			AddProperty("problems", problems).
			Throw()
	}

	return nil
}
