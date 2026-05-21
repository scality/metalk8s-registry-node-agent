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
		return nil, errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Stamp(err)
	}

	file, err := library.GetFile(filePath)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return file, nil
}

// GetPart retrieves the part defined by start/end indexes of a file
// from the root location in the storage based on its fileName.
//
// The caller should close the content reader as early as possible.
func (f *FileSystem) GetPart(fileName string, start int64, end int64) (io.ReadCloser, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return nil, errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Stamp(err)
	}

	file, err := library.GetPart(filePath, start, end)
	if err != nil {
		return nil, errors.Stamp(err)
	}

	return file, nil
}
