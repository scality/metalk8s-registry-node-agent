package di

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externaldownloader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type Container struct {
	// ctx is the base context for the application, and is
	// 	needed to sync shutdown of all dependencies.
	// 	Required by DI pattern.
	ctx context.Context //nolint:containedctx //

	config *config.Environment

	logger *slog.Logger

	filenameChan              chan domain.FileEventDetails
	deleteChan                chan domain.FileEventDetails
	httpExternServer          *http.Server
	httpInternServer          *http.Server
	httpInternClient          *http.Client
	generatedHTTPInternClient *intern.ClientWithResponses
	ExternTLSConfig           *tls.Config
	InternTLSConfig           *tls.Config
	InternTLSClientConfig     *tls.Config
	rootExternAPIPath         string
	rootInternAPIPath         string
	chunkSize                 int64

	solutionArchiveStorage service.StorageProvider
	bucketManager          service.BucketManager
	archiveRemover         service.ArchiveRemover
	archiveReader          service.ArchiveReader
	fileWatcher            service.FileWatcher
	archiveMounter         service.ArchiveMounter
	archiveCleaner         service.ArchiveCleaner
	archiveLister          service.ArchiveLister
	multipartInspector     service.MultipartInspector
	multipartRemover       service.MultipartRemover
	multipartUploader      service.MultipartUploader

	inMemoryArchiveLocker service.LockerUnlocker
	inMemoryBucketLocker  service.LockerUnlocker

	uploadPartHandler              *handler.UploadPart
	downloadSolutionArchiveHandler *handler.DownloadSolutionArchive
	describeSolutionArchiveHandler *handler.DescribeSolutionArchive

	externResolver extern.StrictServerInterface
	internResolver intern.StrictServerInterface

	httpExternalDownloader *externaldownloader.HTTP

	uploadPartUseCase                 *usecase.UploadPart
	initializeSessionUseCase          *usecase.InitializeSession
	removeSolutionArchiveUseCase      *usecase.RemoveSolutionArchive
	removeSessionUseCase              *usecase.RemoveSession
	mountSolutionArchiveUseCase       *usecase.MountSolutionArchive
	unmountSolutionArchiveUseCase     *usecase.UnmountSolutionArchive
	validateSolutionArchiveUseCase    *usecase.ValidateSolutionArchive
	downloadSolutionArchiveUseCase    *usecase.DownloadSolutionArchive
	getExternalSolutionArchiveUseCase *usecase.GetExternalSolutionArchive
	cleanArchivesUseCase              *usecase.CleanArchive
	describeSolutionArchiveUseCase    *usecase.DescribeSolutionArchive
}

func NewContainer(
	ctx context.Context,
	cfg *config.Environment,
	filenameChan chan domain.FileEventDetails,
	deleteChan chan domain.FileEventDetails,
) *Container {
	return &Container{
		ctx:               ctx,
		config:            cfg,
		filenameChan:      filenameChan,
		deleteChan:        deleteChan,
		rootExternAPIPath: cfg.RootExternAPIPath,
		rootInternAPIPath: cfg.RootInternAPIPath,
		chunkSize:         cfg.ChunkSizeMB * 1024 * 1024,
	}
}
