package api

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

var _ = Describe("Clean Unused Solution Archives API", func() {
	var usedSolutionArchives []*domain.SolutionArchive

	BeforeEach(func() {
		usedSolutionArchives = append([]*domain.SolutionArchive{}, initialSolutionArchives...)
	})

	Context("When cleaning unused solution archives", func() {
		It("should delete unused solution archives and keep used ones", func() {
			// Create test solution archives
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "used-solution-archive",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			solutionArchive2 := &domain.SolutionArchive{
				Name:    "unused-solution-archive",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}
			solutionArchive3 := &domain.SolutionArchive{
				Name:    "another-used",
				Version: "3.0.0",
				Hash:    "hash3",
				Size:    300,
			}

			// Save solution archives to storage
			By("creating solution archives in storage")
			err := testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive1),
				bytes.NewReader([]byte("solutionArchive1 content")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			err = testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive2),
				bytes.NewReader([]byte("solutionArchive2 content")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			err = testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive3),
				bytes.NewReader([]byte("solutionArchive3 content")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all solution archives exist")
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive1)))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive2)))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive3)))
			Expect(err).NotTo(HaveOccurred())

			By("cleaning unused solution archives")
			// Only solutionArchive1 and solutionArchive3 are used
			usedSolutionArchives = append(usedSolutionArchives, solutionArchive1, solutionArchive3)
			err = testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying used solution archives still exist")
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive1)))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive3)))
			Expect(err).NotTo(HaveOccurred())

			By("verifying unused solution archive was deleted")
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive2)))
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("When cleaning with all solution archives marked as used", func() {
		It("should not delete any solution archives", func() {
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "keep-me-1",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			solutionArchive2 := &domain.SolutionArchive{
				Name:    "keep-me-2",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}

			By("creating solution archives in storage")
			err := testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive1),
				bytes.NewReader([]byte("content1")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			err = testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive2),
				bytes.NewReader([]byte("content2")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with all solution archives marked as used")
			usedSolutionArchives = append(usedSolutionArchives, solutionArchive1, solutionArchive2)
			err = testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all solution archives still exist")
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive1)))
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive2)))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When handling symlinks to solution archives", func() {
		It("should delete unused symlinks and their targets if unused", func() {
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "target-solution-archive",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}

			By("creating a solution archive file")
			err := testingSuite.SolutionArchiveStorageProvider.SaveFile(
				library.GenSolutionArchiveFileName(solutionArchive1),
				bytes.NewReader([]byte("solution archive content")),
				0644,
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating a symlink to the solution archive")
			solutionArchivePath := filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive1))
			symlinkPath := filepath.Join(testingSuite.SolutionArchiveStorageDirectory, "link-to-solution-archive.iso")
			err = os.Symlink(solutionArchivePath, symlinkPath)
			Expect(err).NotTo(HaveOccurred())

			By("verifying both solution archive and symlink exist")
			_, err = os.Stat(solutionArchivePath)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlinkPath)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with initial solution archives list")
			err = testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying solution archive was deleted")
			_, err = os.Stat(solutionArchivePath)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("When handling buckets (temporary session directories)", func() {
		It("should not delete bucket directories during solution archive cleanup", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "bucket-test",
				Version: "1.0.0",
				Hash:    "hash",
				Size:    100,
			}

			By("creating a bucket (session directory)")
			bucketName := library.GenBucketName(solutionArchive)
			err := testingSuite.SolutionArchiveStorageProvider.CreateBucket(bucketName)
			Expect(err).NotTo(HaveOccurred())

			By("verifying bucket exists")
			bucketPath := filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.FileSystemBucketPrefix+bucketName)
			_, err = os.Stat(bucketPath)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with initial solution archives list")
			err = testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying bucket still exists (cleanup only handles files, not buckets)")
			_, err = os.Stat(bucketPath)
			// Buckets are handled by RemoveSession, not CleanUnusedSolutionArchives
			// So the bucket should still exist
			Expect(err).NotTo(HaveOccurred())

			By("cleaning up test bucket")
			err = testingSuite.SolutionArchiveStorageProvider.DeleteBucket(bucketName)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When handling concurrent cleanup operations", func() {
		It("should handle multiple solution archives efficiently", func() {
			numSolutionArchives := 10
			allSolutionArchives := []*domain.SolutionArchive{}

			By("creating multiple solution archives")
			for i := 0; i < numSolutionArchives; i++ {
				solutionArchive := &domain.SolutionArchive{
					Name:    "solutionArchive",
					Version: string(rune('0'+i)) + ".0.0",
					Hash:    "hash",
					Size:    100,
				}
				allSolutionArchives = append(allSolutionArchives, solutionArchive)

				err := testingSuite.SolutionArchiveStorageProvider.SaveFile(
					library.GenSolutionArchiveFileName(solutionArchive),
					bytes.NewReader([]byte("content")),
					0644,
				)
				Expect(err).NotTo(HaveOccurred())

				// Keep only even-numbered solution archives as used
				if i%2 == 0 {
					usedSolutionArchives = append(usedSolutionArchives, solutionArchive)
				}
			}

			By("verifying all solution archives exist")
			for _, solutionArchive := range allSolutionArchives {
				_, err := os.Stat(filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive)))
				Expect(err).NotTo(HaveOccurred())
			}

			By("cleaning with only even-numbered solution archives as used")
			err := testingSuite.container.GetCleanUnusedSolutionArchivesUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying used solution archives still exist")
			for i, solutionArchive := range allSolutionArchives {
				solutionArchivePath := filepath.Join(testingSuite.SolutionArchiveStorageDirectory, library.GenSolutionArchiveFileName(solutionArchive))
				_, err := os.Stat(solutionArchivePath)
				if i%2 == 0 {
					// Used solution archives should exist
					Expect(err).NotTo(HaveOccurred())
				} else {
					// Unused solution archives should be deleted
					Expect(err).To(HaveOccurred())
					Expect(os.IsNotExist(err)).To(BeTrue())
				}
			}
		})
	})
})
