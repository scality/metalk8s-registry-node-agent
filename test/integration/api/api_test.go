package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rs/zerolog"
	"github.com/scality/metalk8s-registry-node-agent/cmd/config"
	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/infrastructure/di"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
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
	SolutionStorageDirectory        string
	SolutionArchiveStorageProvider  service.StorageProvider

	ExternClientWithResponse *extern.ClientWithResponses
	InternClientWithResponse *intern.ClientWithResponses
}

const (
	httpServerStartupTimeInSeconds = 10
	controlDir                     = ".storageprovider"
	watchedFilesInfoName           = "watched_files_info.json"
)

var (
	ctx          context.Context
	cancel       context.CancelFunc
	testingSuite *TestingSuite

	timeout  = time.Second * 5
	interval = time.Millisecond * 250
)

var initialSolutionArchives []*domain.SolutionArchive

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

	cfg.SolutionArchivesLocation = rootPath + "/archives"
	cfg.SolutionsLocation = rootPath + "/solutions"
	testingSuite = &TestingSuite{
		logger:                          container.GetLogger(),
		container:                       container,
		RootPath:                        rootPath,
		SolutionArchiveStorageDirectory: cfg.SolutionArchivesLocation,
		SolutionStorageDirectory:        cfg.SolutionsLocation,
		ExternClientWithResponse:        externClientWithResponse,
		InternClientWithResponse:        container.GetGeneratedHTTPInternClient(),
		SolutionArchiveStorageProvider:  container.GetFSSolutionArchiveStorage(),
	}

	By("Starting an http Extern Server")
	// Get the server before starting goroutine to avoid race condition
	// during lazy initialization
	httpExternServer := testingSuite.container.GetHTTPExternServer()
	go func() {
		serveErr := httpExternServer.ListenAndServe()
		if serveErr != nil {
			if !errors.Is(serveErr, http.ErrServerClosed) {
				testingSuite.container.GetLogger().Error().Err(serveErr).Msg("http extern server failure during startup")
			}
		}

		testingSuite.container.GetLogger().Info().Msg("http extern server stopped")
	}()

	for range httpServerStartupTimeInSeconds {
		res, err := externHTTPClient.Get(
			fmt.Sprintf("http://localhost%s", httpExternServer.Addr) + "/healthz",
		)
		if err == nil && res.StatusCode == http.StatusOK {
			testingSuite.container.GetLogger().Info().Msg("http extern server is ready")
			break
		}

		time.Sleep(1 * time.Second)
	}

	By("Starting an http Intern Server")
	// Get the server before starting goroutine to avoid race condition
	// during lazy initialization
	httpInternServer := testingSuite.container.GetHTTPInternServer()
	go func() {
		serveErr := httpInternServer.ListenAndServe()
		if serveErr != nil {
			if !errors.Is(serveErr, http.ErrServerClosed) {
				testingSuite.container.GetLogger().Error().Err(serveErr).Msg("http intern server failure during startup")
			}
		}

		testingSuite.container.GetLogger().Info().Msg("http intern server stopped")
	}()

	for range httpServerStartupTimeInSeconds {
		res, err := testingSuite.container.GetHTTPInternClient().Get(
			fmt.Sprintf("http://localhost%s", httpInternServer.Addr) + "/healthz",
		)
		if err == nil && res.StatusCode == http.StatusOK {
			testingSuite.container.GetLogger().Info().Msg("http intern server is ready")
			break
		}

		time.Sleep(1 * time.Second)
	}

	/*
		Initialize with some solution archives
	*/
	initialSolutionArchives = []*domain.SolutionArchive{
		{
			Name:    "artesca-base",
			Version: "4.0.0-preview.1",
			Hash:    "0fac9ac77b2915515aa726a1197536087e69123d49d29d0e397fa4931e7da29e",
			Size:    30,
		},
	}

	for fileName, content := range map[string]string{
		"artesca-base-4.0.0-preview.1.iso": "platform2\nplatform1\nplatform0\n",
	} {
		err := testingSuite.container.
			GetFSSolutionArchiveStorage().
			SaveFile(fileName, bytes.NewReader([]byte(content)), 0644)
		Expect(err).NotTo(HaveOccurred())
	}

	go func() {
		if err := testingSuite.container.GetFSSolutionArchiveStorage().Start(filenameCh); err != nil {
			testingSuite.container.GetLogger().Error().Err(err).Msg("problem starting file system solution archive storage")
		}
	}()

	// Wait for the watched_files_info.json to be created
	Eventually(func() bool {
		_, err := os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, controlDir, watchedFilesInfoName))
		return err == nil
	}, timeout, interval).Should(BeTrue())
})

var _ = AfterSuite(func() {
	By("cleaning with empty used solution archives list")
	err := testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute([]*domain.SolutionArchive{})
	Expect(err).NotTo(HaveOccurred())

	By("verifying all solution archives was deleted")
	fileNames, err := testingSuite.SolutionArchiveStorageProvider.ListFiles()
	Expect(err).NotTo(HaveOccurred())
	Expect(fileNames).To(BeEmpty())

	By("tearing down the test environment")
	cancel()
	defer os.RemoveAll(testingSuite.RootPath) // nolint: errcheck
	err = testingSuite.container.GetHTTPExternServer().Close()
	if err != nil {
		testingSuite.logger.Fatal().Err(err).Msg("http extern server failure during shutdown")
	}
	err = testingSuite.container.GetHTTPInternServer().Close()
	if err != nil {
		testingSuite.logger.Fatal().Err(err).Msg("http intern server failure during shutdown")
	}
})
