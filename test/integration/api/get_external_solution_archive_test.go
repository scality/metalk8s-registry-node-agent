package api

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

// internDownloadURL builds the URL used by the use case to fetch a solution
// archive from another node, hitting the local intern HTTP server.
func internDownloadURL(name, version string) string {
	return fmt.Sprintf(
		"https://localhost%s%s/downloads/%s/%s",
		testingSuite.container.GetHTTPInternServer().Addr,
		testingSuite.container.GetRootInternAPIPath(),
		name,
		version,
	)
}

var _ = Describe("Get External Solution Archive UseCase", func() {
	Context("When the target solution archive already exists in storage", func() {
		It("should be a no-op and return nil", func() {
			By("calling the use case for an already-stored archive")
			err := testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				solutionArchive,
				internDownloadURL(solutionArchive.Name, solutionArchive.Version),
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When downloading a new solution archive from another node", func() {
		It("should successfully download and store the solution archive", func() {
			expectedHash := fmt.Sprintf("%x", sha256.Sum256(solutionArchiveContentBigSize))
			target := &domain.SolutionArchive{
				Name:    "downloaded-archive",
				Version: "1.0.0",
				Hash:    expectedHash,
			}

			By("initializing a session for the target archive")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(target)
			Expect(err).NotTo(HaveOccurred())

			By("calling the use case to download the archive from another node")
			err = testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				target,
				internDownloadURL(solutionArchiveBigSize.Name, solutionArchiveBigSize.Version),
			)
			Expect(err).NotTo(HaveOccurred())

			By("storing the downloaded archive on the file system")
			targetNameVersion := library.GenBucketName(target)
			archiveFile, err := os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					targetNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())
			defer archiveFile.Close() // nolint: errcheck

			storedContent := make([]byte, len(solutionArchiveContentBigSize))
			_, err = archiveFile.Read(storedContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(storedContent).To(Equal(solutionArchiveContentBigSize))

			By("removing the session bucket once the download is complete")
			sessionDir := path.Join(
				testingSuite.SolutionArchiveStorageDirectory,
				library.FileSystemBucketPrefix+targetNameVersion,
			)
			_, err = os.Stat(sessionDir)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the target archive has no initialized session", func() {
		It("should return a 404 not found error", func() {
			target := &domain.SolutionArchive{
				Name:    "no-session-archive",
				Version: "1.0.0",
				Hash:    fmt.Sprintf("%x", sha256.Sum256(solutionArchiveContent)),
			}

			By("calling the use case without initializing a session beforehand")
			err := testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				target,
				internDownloadURL(solutionArchive.Name, solutionArchive.Version),
			)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the session manifest does not match the target archive", func() {
		It("should return an error", func() {
			sessionTarget := &domain.SolutionArchive{
				Name:    "mismatch-archive",
				Version: "1.0.0",
				Hash:    "0000000000000000000000000000000000000000000000000000000000000000",
			}
			differentTarget := &domain.SolutionArchive{
				Name:    "mismatch-archive",
				Version: "1.0.0",
				Hash:    "1111111111111111111111111111111111111111111111111111111111111111",
			}

			By("initializing a session with one hash")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(sessionTarget)
			Expect(err).NotTo(HaveOccurred())

			By("calling the use case with a different hash for the same name/version")
			err = testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				differentTarget,
				internDownloadURL(solutionArchive.Name, solutionArchive.Version),
			)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the source URL points to a non-existent archive", func() {
		It("should return an error", func() {
			target := &domain.SolutionArchive{
				Name:    "missing-source-archive",
				Version: "1.0.0",
				Hash:    fmt.Sprintf("%x", sha256.Sum256(solutionArchiveContent)),
			}

			By("initializing a session for the target archive")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(target)
			Expect(err).NotTo(HaveOccurred())

			By("calling the use case with a URL pointing to a non-existent archive")
			err = testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				target,
				internDownloadURL("non-existent-source-archive", "9.9.9"),
			)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When the source URL is malformed", func() {
		It("should return an error", func() {
			target := &domain.SolutionArchive{
				Name:    "bad-url-archive",
				Version: "1.0.0",
				Hash:    fmt.Sprintf("%x", sha256.Sum256(solutionArchiveContent)),
			}

			By("initializing a session for the target archive")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(target)
			Expect(err).NotTo(HaveOccurred())

			By("calling the use case with an unreachable URL")
			err = testingSuite.container.GetGetExternalSolutionArchiveUseCase().Execute(
				context.TODO(),
				target,
				"https://127.0.0.1:1/api/v1/downloads/nope/0.0.0",
			)
			Expect(err).To(HaveOccurred())
		})
	})
})
