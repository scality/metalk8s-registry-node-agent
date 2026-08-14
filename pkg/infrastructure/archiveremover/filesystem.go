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
		return errors.Wrap(err,
			errors.WithIdentifier(115),
			errors.WithDetail("unexpected error while enforcing naming conventions before deletion"),
			errors.WithProperty("file_name", fileName),
		)
	}

	filePath := filepath.Join(f.solutionArchivesLocation, fileName)
	if err := library.CheckFile(filePath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(116),
			errors.WithDetail("unexpected error while checking solution archive before deletion"),
			errors.WithProperty("file_name", fileName))
	}

	if err := library.DeleteFile(filePath); err != nil {
		return errors.Wrap(err,
			errors.WithIdentifier(117),
			errors.WithDetail("unexpected error while deleting solution archive"),
			errors.WithProperty("file_name", fileName))
	}

	return nil
}
