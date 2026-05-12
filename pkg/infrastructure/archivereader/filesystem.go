package archivereader

import (
	"io"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
) service.ArchiveReader {
	l := logger.With().
		Str("infrastructure", "archive_reader").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
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
		return nil, errors.Wrap(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Wrap(err)
	}

	file, err := library.GetFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return file, nil
}

// GetPart retrieves the part defined by start/end indexes of a file
// from the root location in the storage based on its fileName.
//
// The caller should close the content reader as early as possible.
func (f *FileSystem) GetPart(fileName string, start int64, end int64) (io.ReadCloser, error) {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return nil, errors.Wrap(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return nil, errors.Wrap(err)
	}

	file, err := library.GetPart(filePath, start, end)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return file, nil
}
