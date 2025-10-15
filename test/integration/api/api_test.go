package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rs/zerolog"
	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/di"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/service"
	"github.com/scality/metalk8s-registry-node-agent/test/utils"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type TestingSuite struct {
	logger                          *zerolog.Logger
	container                       *di.Container
	RootPath                        string
	SolutionArchiveStorageDirectory string
	SolutionArchiveStorageProvider  service.StorageProvider

	*extern.ClientWithResponses
}

const (
	httpServerStartupTimeInSeconds = 10
)

var (
	ctx          context.Context
	cancel       context.CancelFunc
	testingSuite *TestingSuite
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "API Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	cfg, err := config.NewEnvironment(ctx)
	if err != nil {
		log.Fatal(err) //nolint:revive // This is basically the main function, shut up revive
	}
	filenameCh := make(chan string)
	container := di.NewContainer(ctx, cfg, filenameCh)

	rootPath, err := os.MkdirTemp("/tmp", "test-integration-api_v1_uploads-*")
	if err != nil {
		container.GetLogger().Fatal().Err(err).Msg("failed to create temporary directory")
	}

	// External HTTP Client creation
	externHTTPClient := utils.GetHTTPExternClient()
	externClientWithResponse, err := utils.GetGeneratedHTTPExternClient(cfg.Extern.Addr, cfg.RootExternAPIPath, externHTTPClient)
	if err != nil {
		container.GetLogger().Fatal().Err(err).Msg("failed to create generated http client")
	}

	cfg.SolutionArchivesLocation = rootPath
	testingSuite = &TestingSuite{
		logger:                          container.GetLogger(),
		container:                       container,
		RootPath:                        rootPath,
		SolutionArchiveStorageDirectory: cfg.SolutionArchivesLocation,
		ClientWithResponses:             externClientWithResponse,
		SolutionArchiveStorageProvider:  container.GetFSSolutionArchiveStorage(),
	}

	By("Starting an http Server")
	// Get the server before starting goroutine to avoid race condition
	// during lazy initialization
	httpExternServer := testingSuite.container.GetHTTPExternServer()
	go func() {
		serveErr := httpExternServer.ListenAndServe()
		if serveErr != nil {
			if !errors.Is(serveErr, http.ErrServerClosed) {
				testingSuite.container.GetLogger().Error().Err(serveErr).Msg("http server failure during startup")
			}
		}

		testingSuite.container.GetLogger().Info().Msg("http server stopped")
	}()

	for range httpServerStartupTimeInSeconds {
		res, err := externHTTPClient.Get(
			fmt.Sprintf("http://localhost%s", httpExternServer.Addr) + "/healthz",
		)
		if err == nil && res.StatusCode == http.StatusOK {
			testingSuite.container.GetLogger().Info().Msg("http server is ready")
		}

		time.Sleep(1 * time.Second)
	}
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	defer os.RemoveAll(testingSuite.RootPath) // nolint: errcheck
	err := testingSuite.container.GetHTTPExternServer().Close()
	if err != nil {
		testingSuite.logger.Fatal().Err(err).Msg("http server failure during shutdown")
	}
})
