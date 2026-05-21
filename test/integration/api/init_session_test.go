package api

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

var _ = Describe("Init Session API", func() {
	Context("When initializing a new session", func() {
		It("should successfully respond to the request", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "test1",
				Version: "1.0.0",
				Hash:    ptr.To("1234567890"),
			}
			By("successfully initialize the session")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(ctx, solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("creating the directory structure")
			// Verify directory/files creation
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			solutionArchiveRootDir := path.Join(testingSuite.SolutionArchiveStorageDirectory, library.FileSystemBucketPrefix+solutionArchiveNameVersion)
			_, err = os.Stat(solutionArchiveRootDir)
			Expect(err).NotTo(HaveOccurred())

			// Checking work files
			metaFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartMetaSuffix)
			partsFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartPartsSuffix)
			recipientFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartRecipientSuffix)

			err = library.CheckFile(metaFilePath)
			Expect(err).NotTo(HaveOccurred())

			err = library.CheckFile(partsFilePath)
			Expect(err).NotTo(HaveOccurred())

			err = library.CheckFile(recipientFilePath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When initializing an existing session", func() {
		It("should successfully respond to the request", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "test2",
				Version: "1.0.0",
				Hash:    ptr.To("1234567890"),
			}

			// Create initial session
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(ctx, solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("accepting the session request, but not creating a new one")
			// Create existing session
			_, err = testingSuite.container.GetInitializeSessionUseCase().Execute(ctx, solutionArchive)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When initializing a session with an existing solution archive", func() {
		It("should successfully respond to the resource", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "test3",
				Version: "1.0.0",
				Hash:    ptr.To("1234567890"),
			}

			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			solutionArchiveContent := []byte("hello\ngo\n")
			err := os.WriteFile(path.Join(testingSuite.SolutionArchiveStorageDirectory, solutionArchiveNameVersion+".iso"), solutionArchiveContent, 0644)
			Expect(err).NotTo(HaveOccurred())

			By("accepting the session request, but not creating a new one")
			_, err = testingSuite.container.GetInitializeSessionUseCase().Execute(ctx, solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("not creating a new directory structure")
			// Verify directory/files creation
			solutionArchiveRootDir := path.Join(testingSuite.SolutionArchiveStorageDirectory, library.FileSystemBucketPrefix+solutionArchiveNameVersion)
			_, err = os.Stat(solutionArchiveRootDir)
			Expect(err).To(HaveOccurred())

			// Checking work files
			metaFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartMetaSuffix)
			partsFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartPartsSuffix)
			recipientFilePath := path.Join(solutionArchiveRootDir, solutionArchive.Name+library.FileSystemMultipartRecipientSuffix)

			err = library.CheckFile(metaFilePath)
			Expect(err).To(HaveOccurred())

			err = library.CheckFile(partsFilePath)
			Expect(err).To(HaveOccurred())

			err = library.CheckFile(recipientFilePath)
			Expect(err).To(HaveOccurred())
		})
	})
})
