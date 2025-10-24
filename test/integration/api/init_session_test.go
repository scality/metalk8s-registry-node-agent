package api

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

var _ = Describe("Init Session API", func() {
	Context("When initializing a new session", func() {
		It("should successfully respond to the request", func() {
			artifact := &domain.Artifact{
				Name:    "test1",
				Version: "1.0.0",
				Hash:    "1234567890",
			}
			By("successfully initialize the session")
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("creating the directory structure")
			// Verify directory/files creation
			artifactNameVersion := library.GenBucketName(artifact)
			artifactRootDir := path.Join(testingSuite.RootPath, library.FileSystemBucketPrefix+artifactNameVersion)
			_, err = os.Stat(artifactRootDir)
			Expect(err).NotTo(HaveOccurred())

			// Checking work files
			metaFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartMetaSuffix)
			partsFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartPartsSuffix)
			recipientFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartRecipientSuffix)

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
			artifact := &domain.Artifact{
				Name:    "test2",
				Version: "1.0.0",
				Hash:    "1234567890",
			}

			// Create initial session
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("accepting the session request, but not creating a new one")
			// Create existing session
			_, err = testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When initializing a session with an existing artifact", func() {
		It("should successfully respond to the resource", func() {
			artifact := &domain.Artifact{
				Name:    "test3",
				Version: "1.0.0",
				Hash:    "1234567890",
			}

			artifactNameVersion := library.GenBucketName(artifact)
			artifactContent := []byte("hello\ngo\n")
			err := os.WriteFile(path.Join(testingSuite.RootPath, artifactNameVersion+".iso"), artifactContent, 0644)
			Expect(err).NotTo(HaveOccurred())

			By("accepting the session request, but not creating a new one")
			_, err = testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("not creating a new directory structure")
			// Verify directory/files creation
			artifactRootDir := path.Join(testingSuite.RootPath, library.FileSystemBucketPrefix+artifactNameVersion)
			_, err = os.Stat(artifactRootDir)
			Expect(err).To(HaveOccurred())

			// Checking work files
			metaFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartMetaSuffix)
			partsFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartPartsSuffix)
			recipientFilePath := path.Join(artifactRootDir, artifact.Name+library.FileSystemMultipartRecipientSuffix)

			err = library.CheckFile(metaFilePath)
			Expect(err).To(HaveOccurred())

			err = library.CheckFile(partsFilePath)
			Expect(err).To(HaveOccurred())

			err = library.CheckFile(recipientFilePath)
			Expect(err).To(HaveOccurred())
		})
	})
})
