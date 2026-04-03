package archivesaver

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/scality/go-errors"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
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
) service.ArchiveSaver {
	l := logger.With().
		Str("infrastructure", "archive_saver").
		Str("implementation", "filesystem").
		Logger()
	return &FileSystem{
		logger:                   &l,
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.ArchiveSaver = &FileSystem{}

func (f *FileSystem) SaveFile(
	fileName string,
	content io.Reader,
	perm os.FileMode,
) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil && !errors.Is(err,
		errors.Intercept(domain.ErrNotFound).WithIdentifier(404000).Throw()) {
		return errors.Stamp(err)
	}

	if err := library.SaveFile(filePath, content, perm); err != nil {
		return errors.Stamp(err)
	}

	return nil
}
