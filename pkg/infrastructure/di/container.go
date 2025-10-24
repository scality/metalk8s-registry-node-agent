package di

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/artifactremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/artifactvalidator"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/partuploader"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessioninitializer"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/sessionremover"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/generated"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/handler"
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

	filenameCh          chan string
	httpServer          *http.Server
	httpClient          *http.Client
	generatedHTTPClient *generated.ClientWithResponses
	rootAPIPath         string

	artifactStorage service.StorageProvider

	uploadPartHandler *handler.UploadPart

	resolver generated.StrictServerInterface

	storagePartUploader       *partuploader.Storage
	storageSessionInitializer *sessioninitializer.Storage
	storageArtifactRemover    *artifactremover.Storage
	storageSessionRemover     *sessionremover.Storage
	storageArtifactValidator  *artifactvalidator.Storage

	uploadPartUseCase        *usecase.UploadPart
	initializeSessionUseCase *usecase.InitializeSession
	removeArtifactUseCase    *usecase.RemoveArtifact
	removeSessionUseCase     *usecase.RemoveSession
	validateArtifactUseCase  *usecase.ValidateArtifact
}

func NewContainer(ctx context.Context, cfg *config.Environment, filenameCh chan string, rootAPIPath string) *Container {
	return &Container{
		baseCtx:     ctx,
		config:      cfg,
		filenameCh:  filenameCh,
		rootAPIPath: rootAPIPath,
	}
}
