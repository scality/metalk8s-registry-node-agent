package api

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Describe Archive API", func() {
	Context("When describing an existing solution archive", func() {
		It("should return the size of the solution archive", func() {
			By("successfully describing the solution archive")
			res, err := testingSuite.InternClientWithResponse.DescribeSolutionArchiveWithResponse(
				context.TODO(),
				solutionArchive.Name,
				solutionArchive.Version,
			)
			Expect(err).NotTo(HaveOccurred())

			By("returning a documented http/200 response with Content-Length")
			Expect(res.HTTPResponse.StatusCode).To(Equal(200))
			Expect(res.HTTPResponse.Header.Get("Content-Length")).To(Equal("30"))
		})
	})

	Context("When describing a non-existent solution archive", func() {
		It("should return a 404 not found error", func() {
			By("attempting to describe a solution archive that doesn't exist")
			// Use raw client because HEAD responses have no body,
			// and the generated WithResponse parser fails on empty JSON
			res, err := testingSuite.InternClientWithResponse.DescribeSolutionArchive(
				context.TODO(),
				"non-existent-archive",
				"1.0.0",
			)
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close() //nolint:errcheck

			By("returning a documented http/404 response")
			Expect(res.StatusCode).To(Equal(404))
		})
	})

	Context("When describing with a non-existent version", func() {
		It("should return a 404 not found error", func() {
			By("attempting to describe an existing archive name with wrong version")
			res, err := testingSuite.InternClientWithResponse.DescribeSolutionArchive(
				context.TODO(),
				"artesca-base",
				"99.99.99",
			)
			Expect(err).NotTo(HaveOccurred())
			defer res.Body.Close() //nolint:errcheck

			By("returning a documented http/404 response")
			Expect(res.StatusCode).To(Equal(404))
		})
	})
})
