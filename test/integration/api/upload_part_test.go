package api

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/extern"
)

const uploadPartTestString = "\x00\nIs there any interesting thing here\n"
const uploadCompleteTestString = "platform0\nplatform1\nplatform2\n\x00"
const uploadMultipleCompleteTestString = "platform0\nplatform1\nplatform2\nplatform3\n\x00"

var _ = Describe("Upload Part API", func() {
	Context("When uploading a new part of an solution archive", func() {
		It("should successfully upload and store the chunk", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "artesca-base",
				Version: "3.0.0-preview.2",
				Hash:    ptr.To("sha"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("successfully upload the chunk")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				"3.0.0-preview.2",
				&extern.UploadChunkParams{
					ContentRange: "bytes 0-37/200",
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
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			recipientFile, err := os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					library.FileSystemBucketPrefix+solutionArchiveNameVersion,
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

	Context("When uploading a new complete solution archive", func() {
		It("should successfully upload the chunks and aggregate the solution archive", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.2-tiny",
				Hash:    ptr.To("a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f0"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks and responding adequately")
			var resUpl *extern.UploadChunkResponse
			for i := range 3 {
				resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					"127.0.2-tiny",
					&extern.UploadChunkParams{
						ContentRange: fmt.Sprintf("bytes %d-%d/30", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			}

			By("returning a documented http/200 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(*resUpl.JSON200.IsCompleted).To(BeTrue())

			By("aggregating and storing the solution archive")
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			solutionArchiveFile, err := os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					solutionArchiveNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())

			defer solutionArchiveFile.Close() // nolint: errcheck

			solutionArchiveBytes := make([]byte, 31)
			_, err = solutionArchiveFile.Read(solutionArchiveBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadCompleteTestString).To(Equal(string(solutionArchiveBytes)))

			By("removing the session")
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

	Context("When uploading a chunk without an opened session", func() {
		It("should fail the process", func() {
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				"3.0.0-preview.3",
				&extern.UploadChunkParams{
					ContentRange: fmt.Sprintf("bytes 0-%d/200", len(uploadPartTestString)-1),
				},
				"application/octet-stream",
				bytes.NewReader([]byte(uploadPartTestString)),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning an http/404 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(404))
		})
	})

	Context("When uploading a new complete solution archive with a wrong checksum", func() {
		It("should fail the process", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.3-tiny",
				Hash:    ptr.To("a2c60bdd4a4fd806fe368bacc30819173ecb9d5f109127bf35dfeaa927b275f1"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks")
			var resUpl *extern.UploadChunkResponse
			// For first uploads (up to the penultimate)
			for i := range 3 {
				resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					"127.0.3-tiny",
					&extern.UploadChunkParams{
						ContentRange: fmt.Sprintf("bytes %d-%d/30", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())

				By("returning an appropriate http/code")
				// The last chunk will return an error due to the bad checksum
				expected := 200
				if i == 2 {
					expected = 422
				}
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(expected))
			}

			By("returning an http/422 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(422))
			Expect(*resUpl.ApplicationproblemJSON422.Code).To(Equal("422-238-192-183"))
			Expect(*resUpl.ApplicationproblemJSON422.Detail).To(ContainSubstring(
				"the hash of the recipient file does not match the solution archive metadata",
			))

			By("not storing the solution archive")
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			_, err = os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					solutionArchiveNameVersion+".iso",
				),
			)
			Expect(err).To(HaveOccurred())

			By("generating a new session")
			// Verify directory/files creation
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

	Context("When uploading a new part of an solution archive with a wrong size", func() {
		It("should fail the process", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "artesca-base",
				Version: "3.0.0-preview.4",
				Hash:    ptr.To("sha"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("uploading a chunk whose body length does not match the Content-Range")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"artesca-base",
				"3.0.0-preview.4",
				&extern.UploadChunkParams{
					ContentRange: "bytes 0-1/200",
				},
				"application/octet-stream",
				bytes.NewReader([]byte(uploadPartTestString)),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(400))
			Expect(*resUpl.ApplicationproblemJSON400.Code).To(Equal("400"))
			Expect(*resUpl.ApplicationproblemJSON400.Status).To(Equal(int32(400)))
		})
	})

	Context("When uploading a new complete solution archive, with having previously sent wrong parts", func() {
		It("should successfully rewrite the correct chunks and aggregate the solution archive", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.3-small",
				Hash:    ptr.To("c8db76b15eda867f25a4baa791cd14a673944fda1298cc440abe8568f33edef7"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			var resUpl *extern.UploadChunkResponse

			/*
				BEGIN - Simulating previous wrong parts uploads
			*/
			// Invert part#1 and #2 files
			// We send part#2 file with Content-Range of part#1
			resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				ctx,
				"platform",
				"127.0.3-small",
				&extern.UploadChunkParams{
					ContentRange: fmt.Sprintf("bytes %d-%d/40", 1*10, (1*10)+9),
				},
				"application/octet-stream",
				bytes.NewReader(fmt.Appendf(nil, "platform%d\n", 2)),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))

			// We send part#2 file with Content-Range of part#3
			resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				ctx,
				"platform",
				"127.0.3-small",
				&extern.UploadChunkParams{
					ContentRange: fmt.Sprintf("bytes %d-%d/40", 2*10, (2*10)+9),
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
				resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					"127.0.3-small",
					&extern.UploadChunkParams{
						ContentRange: fmt.Sprintf("bytes %d-%d/40", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			}

			By("returning a documented http/200 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(*resUpl.JSON200.IsCompleted).To(BeTrue())

			By("aggregating and storing the solution archive")
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			solutionArchiveFile, err := os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					solutionArchiveNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())

			defer solutionArchiveFile.Close() // nolint: errcheck

			solutionArchiveBytes := make([]byte, 41)
			_, err = solutionArchiveFile.Read(solutionArchiveBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadMultipleCompleteTestString).To(Equal(string(solutionArchiveBytes)))

			By("removing the session")
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

	Context("When uploading a new complete solution archive with no checksum provided", func() {
		It("should successfully upload the chunks and aggregate the solution archive", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.2-unchecked",
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("successfully uploading all the chunks and responding adequately")
			var resUpl *extern.UploadChunkResponse
			for i := range 3 {
				resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
					ctx,
					"platform",
					"127.0.2-unchecked",
					&extern.UploadChunkParams{
						ContentRange: fmt.Sprintf("bytes %d-%d/30", i*10, (i*10)+9),
					},
					"application/octet-stream",
					bytes.NewReader(fmt.Appendf(nil, "platform%d\n", i)),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			}

			By("returning a documented http/200 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(*resUpl.JSON200.IsCompleted).To(BeTrue())

			By("aggregating and storing the solution archive")
			solutionArchiveNameVersion := library.GenBucketName(solutionArchive)
			solutionArchiveFile, err := os.Open(
				path.Join(
					testingSuite.SolutionArchiveStorageDirectory,
					solutionArchiveNameVersion+".iso",
				),
			)
			Expect(err).NotTo(HaveOccurred())

			defer solutionArchiveFile.Close() // nolint: errcheck

			solutionArchiveBytes := make([]byte, 31)
			_, err = solutionArchiveFile.Read(solutionArchiveBytes)
			Expect(err).NotTo(HaveOccurred())
			Expect(uploadCompleteTestString).To(Equal(string(solutionArchiveBytes)))

			By("removing the session")
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

	Context("When uploading a chunk with an invalid Content-Range format", func() {
		It("should return a 400 bad request error", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.4-badrange",
				Hash:    ptr.To("sha"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("sending a malformed Content-Range header")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"platform",
				"127.0.4-badrange",
				&extern.UploadChunkParams{
					ContentRange: "not-a-valid-range",
				},
				"application/octet-stream",
				bytes.NewReader([]byte("data")),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(400))
			Expect(resUpl.ApplicationproblemJSON400).NotTo(BeNil())
		})
	})

	Context("When uploading a chunk with start > end in Content-Range", func() {
		It("should return a 400 bad request error", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.4-invertedrange",
				Hash:    ptr.To("sha"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("sending a Content-Range where start > end")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"platform",
				"127.0.4-invertedrange",
				&extern.UploadChunkParams{
					ContentRange: "bytes 20-10/200",
				},
				"application/octet-stream",
				bytes.NewReader([]byte("data")),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(400))
			Expect(resUpl.ApplicationproblemJSON400).NotTo(BeNil())
		})
	})

	Context("When uploading a chunk without the version parameter", func() {
		It("should return a 400 bad request error", func() {
			By("sending an upload without the version parameter")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"platform",
				"",
				&extern.UploadChunkParams{
					ContentRange: "bytes 0-9/100",
				},
				"application/octet-stream",
				bytes.NewReader([]byte("0123456789")),
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(400))
		})
	})

	Context("When uploading overlapping parts", func() {
		It("should accept the overlapping upload and overwrite the overlap region", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "platform",
				Version: "127.0.4-overlap",
				Hash:    ptr.To("sha"),
			}
			_, err := testingSuite.container.GetInitializeSessionUseCase().Execute(solutionArchive)
			Expect(err).NotTo(HaveOccurred())

			By("uploading a first chunk")
			resUpl, err := testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"platform",
				"127.0.4-overlap",
				&extern.UploadChunkParams{
					ContentRange: "bytes 0-14/30",
				},
				"application/octet-stream",
				bytes.NewReader([]byte("0123456789ABCDE")),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(*resUpl.JSON200.IsCompleted).To(BeFalse())

			By("uploading an overlapping second chunk")
			resUpl, err = testingSuite.ExternClientWithResponse.UploadChunkWithBodyWithResponse(
				context.TODO(),
				"platform",
				"127.0.4-overlap",
				&extern.UploadChunkParams{
					ContentRange: "bytes 10-29/30",
				},
				"application/octet-stream",
				bytes.NewReader([]byte("abcdefghijklmnopqrst")),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resUpl.HTTPResponse.StatusCode).To(Equal(200))

			By("having tracked both uploaded chunks")
			Expect(*resUpl.JSON200.UploadedChunks).To(HaveLen(2))
		})
	})
})
