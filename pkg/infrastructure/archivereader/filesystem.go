package archivereader

import (
	"io"
	"log/slog"
	"path/filepath"

	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *slog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *slog.Logger,
	solutionArchivesLocation string,
) service.ArchiveReader {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "archive_reader"),
			slog.String("implementation", "filesystem"),
		),
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.ArchiveReader = &FileSystem{}

// GetFile retrieves the content of a file from the root location in the
// storage based on its fileName.
//
// The caller should close the content reader as early as possible.
func (f *FileSystem) GetFile(fileName string) (io.ReadCloser, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(109),
			errors.WithDetail("unexpected error while enforcing naming conventions before getting solution archive content"),
			errors.WithProperty("file_name", fileName),
		)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(110),
			errors.WithDetail("unexpected error while checking solution archive before getting content"),
			errors.WithProperty("file_name", fileName),
		)
	}

	file, err := library.GetFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(111),
			errors.WithDetail("unexpected error while getting solution archive content"),
			errors.WithProperty("file_name", fileName),
		)
	}

	return file, nil
}

// GetPart retrieves the part defined by start/end indexes of a file
// from the root location in the storage based on its fileName.
//
// The caller should close the content reader as early as possible.
func (f *FileSystem) GetPart(fileName string, start int64, end int64) (io.ReadCloser, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(112),
			errors.WithDetail(
				"unexpected error while enforcing naming conventions before getting solution archive part content",
			),
			errors.WithProperty("file_name", fileName),
			errors.WithProperty("start", start),
			errors.WithProperty("end", end),
		)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(113),
			errors.WithDetail("unexpected error while checking solution archive before getting part content"),
			errors.WithProperty("file_name", fileName),
			errors.WithProperty("start", start),
			errors.WithProperty("end", end),
		)
	}

	file, err := library.GetPart(filePath, start, end)
	if err != nil {
		return nil, errors.Wrap(err,
			errors.WithIdentifier(114),
			errors.WithDetail("unexpected error while getting solution archive part content"),
			errors.WithProperty("file_name", fileName),
			errors.WithProperty("start", start),
			errors.WithProperty("end", end),
		)
	}

	return file, nil
}
