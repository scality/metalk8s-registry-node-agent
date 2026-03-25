package di

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/bucketmanager"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filelister"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filemounter"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filereader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/fileremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/filewatcher"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/externaldownloader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartinspector"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/multipartuploader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/storageprovider"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/usecase"
)

type Container struct {
	baseCtx context.Context //nolint:containedctx //

	config *config.Environment

	logger *zerolog.Logger

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

	storageLifecycle *storageprovider.FileSystem
	storageMu        sync.RWMutex

	filesystemFileReader         *filereader.FileSystem
	filesystemFileLister         *filelister.FileSystem
	filesystemFileRemover        *fileremover.FileSystem
	filesystemFileMounter        *filemounter.FileSystem
	filesystemFileWatcher        *filewatcher.FileSystem
	filesystemBucketManager      *bucketmanager.FileSystem
	filesystemMultipartUploader  *multipartuploader.FileSystem
	filesystemMultipartInspector *multipartinspector.FileSystem

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
	describeSolutionArchiveUseCase    *usecase.DescribeSolutionArchive
	cleanSolutionArchiveUseCase       *usecase.CleanSolutionArchive
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
		chunkSize:         cfg.ChunkSizeMB * 1024 * 1024,
	}
}
