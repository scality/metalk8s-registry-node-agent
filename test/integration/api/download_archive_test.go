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
					ContentRange:    "bytes 0-2/30",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/200 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(200))
			Expect(resDl.Body).NotTo(BeNil())

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
					ContentRange:    "bytes 0-2/4",
				},
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/404 response")
			Expect(resDl.HTTPResponse.StatusCode).To(Equal(404))
			Expect(resDl.ApplicationproblemJSON404).NotTo(BeNil())
		})
	})
})
