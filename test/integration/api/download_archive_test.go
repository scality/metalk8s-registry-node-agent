//nolint:goconst
package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

var _ = Describe("Download Archive API", func() {
	Context("When downloading an existing solution archive", func() {
		It("should successfully download the solution archive", func() {
			solutionArchiveContentHash := sha256.Sum256(solutionArchiveContent[0:3])
			contentDigest := base64.StdEncoding.EncodeToString(solutionArchiveContentHash[:])

			By("successfully downloading the solution archive")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=0-2",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/206 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(206))
			Expect(resDl.Body).NotTo(BeNil())
			Expect(resDl.HTTPResponse.Header.Get("Content-Range")).To(Equal("bytes 0-2/30"))

			By("verifying the downloaded content matches the original solution archive")
			Expect(resDl.Body).To(Equal(solutionArchiveContent[0:3]))

			By("verifying the Content-Digest trailer")
			Expect(resDl.HTTPResponse.Trailer.Get("Content-Digest")).To(Equal("sha-256=:" + contentDigest + ":"))
		})
	})

	Context("When downloading a non-existent solution archive", func() {
		It("should return a 404 not found error", func() {
			By("attempting to download a solution archive that doesn't exist")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				"non-existent-solution-archive",
				"1.0.0",
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=0-2",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/404 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(404))
			Expect(resDl.ApplicationproblemJSON404).NotTo(BeNil())
		})
	})

	Context("When downloading the full content of an existing solution archive", func() {
		It("should return the complete file content", func() {
			solutionArchiveContentHash := sha256.Sum256(solutionArchiveContent[0:30])
			contentDigest := base64.StdEncoding.EncodeToString(solutionArchiveContentHash[:])

			By("successfully downloading the full range")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=0-29",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/206 response with full file range")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(206))
			Expect(resDl.HTTPResponse.Header.Get("Content-Range")).To(Equal("bytes 0-29/30"))

			By("verifying the full content matches")
			Expect(resDl.Body).To(Equal(solutionArchiveContent))

			By("verifying the Content-Digest trailer")
			Expect(resDl.HTTPResponse.Trailer.Get("Content-Digest")).To(Equal("sha-256=:" + contentDigest + ":"))
		})
	})

	Context("When downloading with an invalid Content-Range format", func() {
		It("should return a 400 bad request error", func() {
			By("sending a malformed Content-Range header")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "invalid-range-format",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(400))
			Expect(resDl.ApplicationproblemJSON400).NotTo(BeNil())
		})
	})

	Context("When downloading with a range where start > end", func() {
		It("should return a 400 bad request error", func() {
			By("sending a Content-Range with start greater than end")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=10-5",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/400 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(400))
			Expect(resDl.ApplicationproblemJSON400).NotTo(BeNil())
		})
	})

	Context("When downloading with a range exceeding file size", func() {
		It("should return an error", func() {
			// File is 30 bytes, requesting beyond that
			By("sending a Content-Range beyond the file size")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=0-99",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning an error response")
			Expect(resDl.HTTPResponse.StatusCode).To(BeNumerically(">=", 400))
		})
	})

	Context("When downloading an existing version with a wrong name", func() {
		It("should return a 404 not found error", func() {
			By("attempting to download with the wrong archive name")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				"wrong-name",
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					Range: "bytes=0-2",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/404 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(404))
			Expect(resDl.ApplicationproblemJSON404).NotTo(BeNil())
		})
	})
})
