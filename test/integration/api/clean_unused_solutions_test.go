package api

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/scality/metalk8s-registry-node-agent/pkg/domain"
	"github.com/scality/metalk8s-registry-node-agent/pkg/library"
)

var _ = Describe("Clean Unused Solutions API", func() {
	Context("When cleaning unused solutions", func() {
		It("should delete unused solution directories and keep used ones", func() {
			// Create test solution archives
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "solution-a",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			solutionArchive2 := &domain.SolutionArchive{
				Name:    "solution-b",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}
			solutionArchive3 := &domain.SolutionArchive{
				Name:    "solution-a",
				Version: "3.0.0",
				Hash:    "hash3",
				Size:    300,
			}

			// Create solution directory structure
			By("creating solution directories")
			solutionDir1 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive1))
			solutionDir2 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive2))
			solutionDir3 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive3))

			err := os.MkdirAll(solutionDir1, 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(solutionDir2, 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(solutionDir3, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all solution directories exist")
			_, err = os.Stat(solutionDir1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(solutionDir2)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(solutionDir3)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning unused solutions")
			// Only solutionArchive1 and solutionArchive3 are used
			usedSolutionArchives := []*domain.SolutionArchive{solutionArchive1, solutionArchive3}
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying used solution directories still exist")
			_, err = os.Stat(solutionDir1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(solutionDir3)
			Expect(err).NotTo(HaveOccurred())

			By("verifying unused solution directory was deleted")
			_, err = os.Stat(solutionDir2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("cleaning up test directories")
			os.RemoveAll(filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive1.Name)) // nolint: errcheck
		})
	})

	Context("When cleaning with no used solutions", func() {
		It("should delete all solution directories", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "unused-solution",
				Version: "1.0.0",
				Hash:    "hash",
				Size:    100,
			}

			By("creating a solution directory")
			solutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive))
			err := os.MkdirAll(solutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("verifying solution directory exists")
			_, err = os.Stat(solutionDir)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with empty used solution archives list")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{})
			Expect(err).NotTo(HaveOccurred())

			By("verifying solution directory was deleted")
			_, err = os.Stat(solutionDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying parent name directory was also cleaned up")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive.Name)
			_, err = os.Stat(nameDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("When cleaning with all solutions marked as used", func() {
		It("should not delete any solution directories", func() {
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "keep-solution-1",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			solutionArchive2 := &domain.SolutionArchive{
				Name:    "keep-solution-2",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}

			By("creating solution directories")
			solutionDir1 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive1))
			solutionDir2 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive2))

			err := os.MkdirAll(solutionDir1, 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(solutionDir2, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with all solutions marked as used")
			usedSolutionArchives := []*domain.SolutionArchive{solutionArchive1, solutionArchive2}
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all solution directories still exist")
			_, err = os.Stat(solutionDir1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(solutionDir2)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning up test directories")
			os.RemoveAll(filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive1.Name)) // nolint: errcheck
			os.RemoveAll(filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive2.Name)) // nolint: errcheck
		})
	})

	Context("When cleaning solutions with unexpected files", func() {
		It("should delete unexpected files in solution directories", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "clean-solution",
				Version: "1.0.0",
				Hash:    "hash",
				Size:    100,
			}

			By("creating a solution directory")
			solutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive))
			err := os.MkdirAll(solutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating unexpected files in solution location root")
			unexpectedFile1 := filepath.Join(testingSuite.SolutionStorageDirectory, "unexpected-file.txt")
			err = os.WriteFile(unexpectedFile1, []byte("unexpected content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("creating unexpected files in name directory")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive.Name)
			unexpectedFile2 := filepath.Join(nameDir, "unexpected-file-in-name.txt")
			err = os.WriteFile(unexpectedFile2, []byte("unexpected content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("verifying unexpected files exist")
			_, err = os.Stat(unexpectedFile1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(unexpectedFile2)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with the solution marked as used")
			usedSolutionArchives := []*domain.SolutionArchive{solutionArchive}
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying solution directory still exists")
			_, err = os.Stat(solutionDir)
			Expect(err).NotTo(HaveOccurred())

			By("verifying unexpected files were deleted")
			_, err = os.Stat(unexpectedFile1)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(unexpectedFile2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("cleaning up test directories")
			os.RemoveAll(filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive.Name)) // nolint: errcheck
		})
	})

	Context("When cleaning solutions with multiple versions of the same name", func() {
		It("should keep used versions and delete unused ones", func() {
			solutionArchive1 := &domain.SolutionArchive{
				Name:    "multi-version",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			solutionArchive2 := &domain.SolutionArchive{
				Name:    "multi-version",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}
			solutionArchive3 := &domain.SolutionArchive{
				Name:    "multi-version",
				Version: "3.0.0",
				Hash:    "hash3",
				Size:    300,
			}

			By("creating solution directories for all versions")
			solutionDir1 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive1))
			solutionDir2 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive2))
			solutionDir3 := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive3))

			err := os.MkdirAll(solutionDir1, 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(solutionDir2, 0755)
			Expect(err).NotTo(HaveOccurred())
			err = os.MkdirAll(solutionDir3, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with only version 1.0.0 and 3.0.0 as used")
			usedSolutionArchives := []*domain.SolutionArchive{solutionArchive1, solutionArchive3}
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute(usedSolutionArchives)
			Expect(err).NotTo(HaveOccurred())

			By("verifying used versions still exist")
			_, err = os.Stat(solutionDir1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(solutionDir3)
			Expect(err).NotTo(HaveOccurred())

			By("verifying unused version was deleted")
			_, err = os.Stat(solutionDir2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying parent name directory still exists (has used versions)")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive1.Name)
			_, err = os.Stat(nameDir)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning up test directories")
			os.RemoveAll(nameDir) // nolint: errcheck
		})
	})

	Context("When cleaning solutions with files in solution root", func() {
		It("should delete files and symlinks directly in solution root", func() {
			By("creating files directly in solution root")
			fakeFile := filepath.Join(testingSuite.SolutionStorageDirectory, "fake_solution.txt")
			err := os.WriteFile(fakeFile, []byte("test_1"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("creating symlinks directly in solution root")
			symlink1 := filepath.Join(testingSuite.SolutionStorageDirectory, "0fake_solution_link.txt")
			err = os.Symlink(fakeFile, symlink1)
			Expect(err).NotTo(HaveOccurred())

			symlink2 := filepath.Join(testingSuite.SolutionStorageDirectory, "zfake_solution_link.txt")
			err = os.Symlink(fakeFile, symlink2)
			Expect(err).NotTo(HaveOccurred())

			By("verifying files and symlinks exist")
			_, err = os.Stat(fakeFile)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlink1) // Use Lstat to not follow symlink
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlink2)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with empty used solution archives list")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{})
			Expect(err).NotTo(HaveOccurred())

			By("verifying all unexpected files and symlinks were deleted")
			_, err = os.Stat(fakeFile)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Lstat(symlink1)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Lstat(symlink2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("When cleaning solutions with files as solution directories", func() {
		It("should delete files and symlinks in name directories", func() {
			By("creating a name directory with a file")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, "solution_A")
			err := os.MkdirAll(nameDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			versionFile := filepath.Join(nameDir, "version_as_file.txt")
			err = os.WriteFile(versionFile, []byte("test_2"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("creating another name directory with valid version directory")
			nameDir2 := filepath.Join(testingSuite.SolutionStorageDirectory, "solution_B")
			validVersion := filepath.Join(nameDir2, "1.25.5")
			err = os.MkdirAll(validVersion, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating symlinks pointing to version directories")
			symlink1 := filepath.Join(nameDir2, "1.25.4")
			err = os.Symlink(validVersion, symlink1)
			Expect(err).NotTo(HaveOccurred())

			symlink2 := filepath.Join(nameDir2, "1.25.6")
			err = os.Symlink(validVersion, symlink2)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all items exist")
			_, err = os.Stat(versionFile)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlink1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlink2)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with empty used solution archives list")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{})
			Expect(err).NotTo(HaveOccurred())

			By("verifying files in name directories were deleted")
			_, err = os.Stat(versionFile)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying symlinks in name directories were deleted")
			_, err = os.Lstat(symlink1)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Lstat(symlink2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying valid version directory was deleted (unused)")
			_, err = os.Stat(validVersion)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying empty name directories were cleaned up")
			_, err = os.Stat(nameDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(nameDir2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("When cleaning solutions with files in version directories", func() {
		It("should handle files, directories, and symlinks inside mounted solutions", func() {
			solutionArchive := &domain.SolutionArchive{
				Name:    "solution_C",
				Version: "3.2.1",
				Hash:    "hash",
				Size:    100,
			}

			By("creating solution directory structure")
			solutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(solutionArchive))
			err := os.MkdirAll(solutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating a file inside the version directory")
			fileInVersion := filepath.Join(solutionDir, "version_as_file.txt")
			err = os.WriteFile(fileInVersion, []byte("test_3"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("creating subdirectories inside the version directory")
			subdir1 := filepath.Join(solutionDir, "repertoire1")
			err = os.MkdirAll(subdir1, 0755)
			Expect(err).NotTo(HaveOccurred())

			subdir2 := filepath.Join(solutionDir, "repertoire2")
			err = os.MkdirAll(subdir2, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating a symlink inside the version directory")
			symlink := filepath.Join(solutionDir, "repertoire3")
			err = os.Symlink(subdir2, symlink)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all items exist")
			_, err = os.Stat(fileInVersion)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(subdir1)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(subdir2)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(symlink)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with empty used solution archives list (solution is unused)")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the entire solution directory was deleted")
			_, err = os.Stat(solutionDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying the parent name directory was cleaned up")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, solutionArchive.Name)
			_, err = os.Stat(nameDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should preserve content in used solutions but clean unused ones with complex content", func() {
			usedSolutionArchive := &domain.SolutionArchive{
				Name:    "solution_D",
				Version: "1.0.0",
				Hash:    "hash1",
				Size:    100,
			}
			unusedSolutionArchive := &domain.SolutionArchive{
				Name:    "solution_D",
				Version: "2.0.0",
				Hash:    "hash2",
				Size:    200,
			}

			By("creating used solution with complex content")
			usedSolutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(usedSolutionArchive))
			err := os.MkdirAll(usedSolutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			usedFile := filepath.Join(usedSolutionDir, "important_file.txt")
			err = os.WriteFile(usedFile, []byte("important data"), 0644)
			Expect(err).NotTo(HaveOccurred())

			usedSubdir := filepath.Join(usedSolutionDir, "important_dir")
			err = os.MkdirAll(usedSubdir, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating unused solution with complex content")
			unusedSolutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(unusedSolutionArchive))
			err = os.MkdirAll(unusedSolutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			unusedFile := filepath.Join(unusedSolutionDir, "unwanted_file.txt")
			err = os.WriteFile(unusedFile, []byte("unwanted data"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with only the first version as used")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{usedSolutionArchive})
			Expect(err).NotTo(HaveOccurred())

			By("verifying used solution and its content still exist")
			_, err = os.Stat(usedSolutionDir)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(usedFile)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(usedSubdir)
			Expect(err).NotTo(HaveOccurred())

			By("verifying unused solution was completely removed")
			_, err = os.Stat(unusedSolutionDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying parent name directory still exists (has used version)")
			nameDir := filepath.Join(testingSuite.SolutionStorageDirectory, usedSolutionArchive.Name)
			_, err = os.Stat(nameDir)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning up test directories")
			os.RemoveAll(nameDir) // nolint: errcheck
		})
	})

	Context("Combined edge cases", func() {
		It("should handle all levels of issues simultaneously", func() {
			usedSolutionArchive := &domain.SolutionArchive{
				Name:    "valid_solution",
				Version: "1.0.0",
				Hash:    "hash",
				Size:    100,
			}

			By("creating a valid used solution")
			validSolutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(usedSolutionArchive))
			err := os.MkdirAll(validSolutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			By("creating Level I issues")
			rootFile := filepath.Join(testingSuite.SolutionStorageDirectory, "root_level_file.txt")
			err = os.WriteFile(rootFile, []byte("root"), 0644)
			Expect(err).NotTo(HaveOccurred())

			rootSymlink := filepath.Join(testingSuite.SolutionStorageDirectory, "root_level_link.txt")
			err = os.Symlink(rootFile, rootSymlink)
			Expect(err).NotTo(HaveOccurred())

			By("creating Level II issues")
			invalidNameDir := filepath.Join(testingSuite.SolutionStorageDirectory, "invalid_solution")
			err = os.MkdirAll(invalidNameDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			nameFile := filepath.Join(invalidNameDir, "name_level_file.txt")
			err = os.WriteFile(nameFile, []byte("name"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("creating an unused solution with complex content (Level III)")
			unusedSolutionArchive := &domain.SolutionArchive{
				Name:    "invalid_solution",
				Version: "5.0.0",
				Hash:    "hash2",
				Size:    200,
			}
			unusedSolutionDir := filepath.Join(testingSuite.SolutionStorageDirectory, library.GenSolutionDirName(unusedSolutionArchive))
			err = os.MkdirAll(unusedSolutionDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			complexFile := filepath.Join(unusedSolutionDir, "complex.txt")
			err = os.WriteFile(complexFile, []byte("complex"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("verifying all items exist before cleanup")
			_, err = os.Stat(rootFile)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Lstat(rootSymlink)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(nameFile)
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat(unusedSolutionDir)
			Expect(err).NotTo(HaveOccurred())

			By("cleaning with only valid solution marked as used")
			err = testingSuite.container.GetCleanUnusedSolutionsUseCase().Execute([]*domain.SolutionArchive{usedSolutionArchive})
			Expect(err).NotTo(HaveOccurred())

			By("verifying valid solution still exists")
			_, err = os.Stat(validSolutionDir)
			Expect(err).NotTo(HaveOccurred())

			By("verifying Level I issues were cleaned")
			_, err = os.Stat(rootFile)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Lstat(rootSymlink)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying Level II issues were cleaned")
			_, err = os.Stat(nameFile)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying Level III (unused solution) was cleaned")
			_, err = os.Stat(unusedSolutionDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("verifying invalid name directory was cleaned up")
			_, err = os.Stat(invalidNameDir)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			By("cleaning up test directories")
			os.RemoveAll(filepath.Join(testingSuite.SolutionStorageDirectory, usedSolutionArchive.Name)) // nolint: errcheck
		})
	})
})
