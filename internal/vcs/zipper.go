package vcs

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateZip compresses the source directory into a target zip file path.
func CreateZip(sourceDir, targetZipPath string, ignoredDirs []string) error {
	zipFile, err := os.Create(targetZipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	ignoreMap := make(map[string]bool)
	for _, d := range ignoredDirs {
		ignoreMap[d] = true
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if ignoreMap[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Use standard forward slashes for ZIP format
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
