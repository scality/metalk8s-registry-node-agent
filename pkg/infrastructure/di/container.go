package di

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivedownloader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivemounter"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archiveremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archiveunmounter"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/archivevalidator"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externaldownloader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externalsolutionarchivegetter"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/partuploader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessioninitializer"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessionremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/solutionarchivecleaner"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type Container struct {
	// This is the base context for the application, and is
	// 	needed to sync shutdown of all dependencies.
	// 	Required by DI pattern.
	baseCtx context.Context //nolint:containedctx //

	config *config.Environment

	logger *zerolog.Logger

	filenameChan              chan domain.FileEventDetails
	deleteChan                chan domain.FileEventDetails
	httpExternServer          *http.Server
	httpInternServer          *http.Server
	httpInternClient          *http.Client
	generatedHTTPInternClient *intern.ClientWithResponses
	rootExternAPIPath         string
	rootInternAPIPath         string

	solutionArchiveStorage service.StorageProvider

	uploadPartHandler              *handler.UploadPart
	downloadSolutionArchiveHandler *handler.DownloadSolutionArchive

	externResolver extern.StrictServerInterface
	internResolver intern.StrictServerInterface

	httpExternalDownloader               *externaldownloader.HTTP
	storagePartUploader                  *partuploader.Storage
	storageSessionInitializer            *sessioninitializer.Storage
	storageSolutionArchiveRemover        *archiveremover.Storage
	storageSessionRemover                *sessionremover.Storage
	storageSolutionArchiveValidator      *archivevalidator.Storage
	storageSolutionArchiveDownloader     *archivedownloader.Storage
	storageExternalSolutionArchiveGetter *externalsolutionarchivegetter.Storage
	storageSolutionArchiveMounter        *archivemounter.Storage
	storageSolutionArchiveUnmounter      *archiveunmounter.Storage
	solutionArchiveCleaner               *solutionarchivecleaner.FileSystem

	uploadPartUseCase                 *usecase.UploadPart
	initializeSessionUseCase          *usecase.InitializeSession
	removeSolutionArchiveUseCase      *usecase.RemoveSolutionArchive
	removeSessionUseCase              *usecase.RemoveSession
	mountSolutionArchiveUseCase       *usecase.MountSolutionArchive
	unmountSolutionArchiveUseCase     *usecase.UnmountSolutionArchive
	validateSolutionArchiveUseCase    *usecase.ValidateSolutionArchive
	downloadSolutionArchiveUseCase    *usecase.DownloadSolutionArchive
	getExternalSolutionArchiveUseCase *usecase.GetExternalSolutionArchive
}

func NewContainer(
	ctx context.Context,
	cfg *config.Environment,
	filenameChan chan domain.FileEventDetails,
	deleteChan chan domain.FileEventDetails,
) *Container {
	return &Container{
		baseCtx:           ctx,
		config:            cfg,
		filenameChan:      filenameChan,
		deleteChan:        deleteChan,
		rootExternAPIPath: cfg.RootExternAPIPath,
		rootInternAPIPath: cfg.RootInternAPIPath,
	}
}
