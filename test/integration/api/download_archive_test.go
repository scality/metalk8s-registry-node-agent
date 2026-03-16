package api

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/presentation/http/intern"
)

var _ = Describe("Download Archive API", func() {
	Context("When downloading an existing solution archive", func() {
		It("should successfully download the solution archive", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "artesca-base",
				Version: "4.0.0-preview.1",
				Hash:    "0fac9ac77b2915515aa726a1197536087e69123d49d29d0e397fa4931e7da29e",
			}
			solutionArchiveContent := []byte("platform2\nplatform1\nplatform0\n")

			By("successfully downloading the solution archive")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				"artesca-base",
				solutionArchive.Version,
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: solutionArchive.Hash,
					Range:           "bytes=0-2",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/206 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(206))
			Expect(resDl.Body).NotTo(BeNil())
			Expect(resDl.HTTPResponse.Header.Get("Content-Range")).To(Equal("bytes 0-2/30"))

			By("verifying the downloaded content matches the original solution archive")
			Expect(resDl.Body).To(Equal(solutionArchiveContent[0:3]))
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
					XSha256Checksum: "sha",
					Range:           "bytes=0-2",
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
			// "artesca-base-4.0.0-preview.1.iso" has content "platform2\nplatform1\nplatform0\n" (30 bytes)
			solutionArchiveContent := []byte("platform2\nplatform1\nplatform0\n")

			By("successfully downloading the full range")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				"artesca-base",
				"4.0.0-preview.1",
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: "0fac9ac77b2915515aa726a1197536087e69123d49d29d0e397fa4931e7da29e",
					Range:           "bytes=0-29",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/206 response with full file range")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(206))
			Expect(resDl.HTTPResponse.Header.Get("Content-Range")).To(Equal("bytes 0-29/30"))

			By("verifying the full content matches")
			Expect(resDl.Body).To(Equal(solutionArchiveContent))
		})
	})

	Context("When downloading with an invalid Content-Range format", func() {
		It("should return a 400 bad request error", func() {
			By("sending a malformed Content-Range header")
			resDl, err := testingSuite.InternClientWithResponse.DownloadSolutionArchiveWithResponse(
				context.TODO(),
				"artesca-base",
				"4.0.0-preview.1",
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: "sha",
					Range:           "invalid-range-format",
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
				"artesca-base",
				"4.0.0-preview.1",
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: "sha",
					Range:           "bytes=10-5",
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
				"artesca-base",
				"4.0.0-preview.1",
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: "sha",
					Range:           "bytes=0-99",
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
				"4.0.0-preview.1",
				&intern.DownloadSolutionArchiveParams{
					XSha256Checksum: "sha",
					Range:           "bytes=0-2",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/404 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(404))
			Expect(resDl.ApplicationproblemJSON404).NotTo(BeNil())
		})
	})
})
