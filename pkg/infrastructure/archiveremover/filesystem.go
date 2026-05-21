package archiveremover

import (
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
) service.ArchiveRemover {
	return &FileSystem{
		logger: logger.With(
			slog.String("infrastructure", "archive_remover"),
			slog.String("implementation", "filesystem"),
		),
		solutionArchivesLocation: solutionArchivesLocation,
	}
}

var _ service.ArchiveRemover = &FileSystem{}

// DeleteFile deletes a file from the root location in the storage based on its fileName.
func (f *FileSystem) DeleteFile(
	fileName string,
) error {
	if err := library.EnforceNamingConventions(fileName); err != nil {
		return errors.Stamp(err)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	if err := library.DeleteFile(filePath); err != nil {
		return errors.Stamp(err)
	}

	return nil
}
