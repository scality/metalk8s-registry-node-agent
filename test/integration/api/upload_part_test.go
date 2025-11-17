package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

const uploadPartTestString = "\x00\nIs there any interesting thing here\n"
const uploadCompleteTestString = "platform0\nplatform1\nplatform2\n\x00"
const uploadMultipleCompleteTestString = "platform0\nplatform1\nplatform2\nplatform3\n\x00"

var _ = Describe("Upload Part API", func() {
	Context("When uploading a new part of an artifact", func() {
		It("should successfully upload and store the chunk", func() {
			artifact := &domain.Artifact{
				Name:    "artesca-base",
				Version: "3.0.0-preview.2",
				Hash:    "sha",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("successfully upload the chunk")
			resUpl, err := testingSuite.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				&extern.UploadChunkParams{
					XTargetVersion:  "3.0.0-preview.2",
					XSha256Checksum: "sha",
					ContentRange:    "bytes 0-37/200",
				},
				"application/octet-stream",
				bytes.NewReader([]byte(uploadPartTestString)),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/200 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(*resUpl.JSON200.IsCompleted).To(BeFalse())
			Expect(*resUpl.JSON200.Version).To(Equal("3.0.0-preview.2"))
			Expect(*resUpl.JSON200.Sha256sum).To(Equal("sha"))
			Expect(*resUpl.JSON200.UploadedChunks).To(HaveLen(1))

			By("storing the part on the file system")
			artifactNameVersion := library.GenBucketName(artifact)
			recipientFile, err := os.Open(
				path.Join(
					testingSuite.ArtifactStorageDirectory,
					library.FileSystemBucketPrefix+artifactNameVersion,
					"artesca-base.recipient",
				),
			)

			Expect(err).NotTo(HaveOccurred())

			defer recipientFile.Close() // nolint: errcheck

			chunkBytes := make([]byte, 38)
			_, err = recipientFile.Read(chunkBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadPartTestString).To(Equal(string(chunkBytes)))
		})
	})

	Context("When uploading a new complete artifact", func() {
		It("should successfully upload the chunks and aggregate the artifact", func() {
			artifact := &domain.Artifact{
				Name:    "platform",
				Version: "127.0.2-tiny",
				Hash:    "a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f0",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks and responding adequately")
			var resUpl *extern.UploadChunkResponse
			for i := range 3 {
				resUpl, err = testingSuite.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					&extern.UploadChunkParams{
						XTargetVersion:  "127.0.2-tiny",
						XSha256Checksum: "a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f0",
						ContentRange:    fmt.Sprintf("bytes %d-%d/30", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			}

			By("returning a documented http/200 response")
			Expect(*resUpl.JSON200.IsCompleted).To(BeTrue())

			By("aggregating and storing the artifact")
			artifactNameVersion := library.GenBucketName(artifact)
			artifactFile, err := os.Open(
				path.Join(
					testingSuite.RootPath,
					artifactNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())

			defer artifactFile.Close() // nolint: errcheck

			artifactBytes := make([]byte, 31)
			_, err = artifactFile.Read(artifactBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadCompleteTestString).To(Equal(string(artifactBytes)))

			By("removing the session")
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

	Context("When uploading a chunk without an opened session", func() {
		It("should fail the process", func() {
			resUpl, err := testingSuite.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				&extern.UploadChunkParams{
					XTargetVersion:  "3.0.0-preview.3",
					XSha256Checksum: "sha",
					ContentRange:    "bytes 0-19/200",
				},
				"application/octet-stream",
				bytes.NewReader([]byte(uploadPartTestString)),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning an http/404 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(404))
		})
	})

	Context("When uploading a new complete artifact with a wrong checksum", func() {
		It("should fail the process", func() {
			artifact := &domain.Artifact{
				Name:    "platform",
				Version: "127.0.3-tiny",
				Hash:    "a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f1",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks")
			var resUpl *extern.UploadChunkResponse
			// For first uploads (up to the penultimate)
			for i := range 3 {
				resUpl, err = testingSuite.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					&extern.UploadChunkParams{
						XTargetVersion:  "127.0.3-tiny",
						XSha256Checksum: "a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f1",
						ContentRange:    fmt.Sprintf("bytes %d-%d/30", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())

				By("returning an appropriate http/code")
				// The last chunk will return an error due to the bad checksum
				expected := 200
				if i == 2 {
					expected = 400
				}
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(expected))
			}

			By("returning an http/422 response")
			Expect(*resUpl.ApplicationproblemJSON400.Code).To(Equal("422001"))
			Expect(*resUpl.ApplicationproblemJSON400.Status).To(Equal(int32(422)))

			By("not storing the artifact")
			artifactNameVersion := library.GenBucketName(artifact)
			_, err = os.Open(
				path.Join(
					testingSuite.RootPath,
					artifactNameVersion+".iso",
				),
			)
			Expect(err).To(HaveOccurred())

			By("generating a new session")
			// Verify directory/files creation
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

	Context("When uploading a new part of an artifact with a wrong size", func() {
		It("should fail the process", func() {
			artifact := &domain.Artifact{
				Name:    "artesca-base",
				Version: "3.0.0-preview.4",
				Hash:    "sha",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks")
			resUpl, err := testingSuite.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				&extern.UploadChunkParams{
					XTargetVersion:  "3.0.0-preview.4",
					XSha256Checksum: "sha",
					ContentRange:    "bytes 0-1/200",
				},
				"application/octet-stream",
				bytes.NewReader([]byte(uploadPartTestString)),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning an http/500 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(500))
			Expect(*resUpl.ApplicationproblemJSON500.Code).To(Equal("500000"))
			Expect(*resUpl.ApplicationproblemJSON500.Status).To(Equal(int32(500)))
		})
	})

	Context("When uploading a new complete artifact, with having previously sent wrong parts", func() {
		It("should successfully rewrite the correct chunks and aggregate the artifact", func() {
			artifact := &domain.Artifact{
				Name:    "platform",
				Version: "127.0.3-small",
				Hash:    "c8db76b15eda867f25a4baa791cd14a673944fda1298cc440abe8568f33edef7",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(artifact)
			Expect(err).NotTo(HaveOccurred())

			var resUpl *extern.UploadChunkResponse

			/*
				BEGIN - Simulating previous wrong parts uploads
			*/
			// Invert part#1 and #2 files
			// We send part#2 file with Content-Range of part#1
			resUpl, err = testingSuite.UploadChunkWithBodyWithResponse(
				ctx,
				"platform",
				&extern.UploadChunkParams{
					XTargetVersion:  "127.0.3-small",
					XSha256Checksum: "c8db76b15eda867f25a4baa791cd14a673944fda1298cc440abe8568f33edef7",
					ContentRange:    fmt.Sprintf("bytes %d-%d/40", 1*10, (1*10)+9),
				},
				"application/octet-stream",
				bytes.NewReader(fmt.Appendf(nil, "platform%d\n", 2)),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))

			// We send part#2 file with Content-Range of part#3
			resUpl, err = testingSuite.UploadChunkWithBodyWithResponse(
				ctx,
				"platform",
				&extern.UploadChunkParams{
					XTargetVersion:  "127.0.3-small",
					XSha256Checksum: "c8db76b15eda867f25a4baa791cd14a673944fda1298cc440abe8568f33edef7",
					ContentRange:    fmt.Sprintf("bytes %d-%d/40", 2*10, (2*10)+9),
				},
				"application/octet-stream",
				bytes.NewReader(fmt.Appendf(nil, "platform%d\n", 1)),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			/*
				END - Simulating previous wrong parts uploads
			*/

			By("successfully uploading all the chunks and responding adequately")
			// Resend all chunks with correct order
			for i := range 4 {
				resUpl, err = testingSuite.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					&extern.UploadChunkParams{
						XTargetVersion:  "127.0.3-small",
						XSha256Checksum: "c8db76b15eda867f25a4baa791cd14a673944fda1298cc440abe8568f33edef7",
						ContentRange:    fmt.Sprintf("bytes %d-%d/40", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			}

			By("returning a documented http/200 response")
			Expect(*resUpl.JSON200.IsCompleted).To(BeTrue())

			By("aggregating and storing the artifact")
			artifactNameVersion := library.GenBucketName(artifact)
			artifactFile, err := os.Open(
				path.Join(
					testingSuite.RootPath,
					artifactNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())

			defer artifactFile.Close() // nolint: errcheck

			artifactBytes := make([]byte, 41)
			_, err = artifactFile.Read(artifactBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadMultipleCompleteTestString).To(Equal(string(artifactBytes)))

			By("removing the session")
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
