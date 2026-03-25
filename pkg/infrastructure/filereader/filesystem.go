package filereader

import (
	"io"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
)

var _ service.FileReader = &FileSystem{}

type FileSystem struct {
	logger                   *zerolog.Logger
	solutionArchivesLocation string
}

func NewFileSystem(
	logger *zerolog.Logger,
	solutionArchivesLocation string,
) *FileSystem {
	l := logger.With().
		Str("infrastructure", "file_reader").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

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
