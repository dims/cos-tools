package fs

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"cos.googlesource.com/cos/tools.git/src/pkg/utils"
)

const gzipFileExt = ".gz"

// TarFile compresses the file at src to dst.
func TarFile(src, dst string) error {
	args := []string{"cf", dst}
	dirPath := filepath.Dir(src)
	baseName := filepath.Base(src)
	// Add the compression type based on the dst, if the
	// file type is not supported tar using default compression.
	if filepath.Ext(dst) == gzipFileExt {
		args = append(args, "-I", "/bin/gzip")
	}
	// add inputFilePath args
	args = append(args, "-C", dirPath, baseName)
	cmd := exec.Command("tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// TarDir compresses the directory at root to dst.
func TarDir(root, dst string) error {
	args := []string{"cf", dst, "-C", root}
	inputFiles, err := filepath.Glob(filepath.Join(root, "*"))
	if err != nil {
		return err
	}
	var relInputFiles []string
	for _, path := range inputFiles {
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relInputFiles = append(relInputFiles, relPath)
	}
	if relInputFiles == nil {
		relInputFiles = []string{"."}
	}
	args = append(args, relInputFiles...)
	cmd := exec.Command("tar", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ExtractFile decompresses the tar file at inputFile to destDir.
func ExtractFile(inputFile, destDir string) error {
	var reader io.Reader
	fileReader, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer utils.CheckClose(fileReader, "error closing the file reader", &err)
	if filepath.Ext(inputFile) == ".gz" {
		reader, err = gzip.NewReader(fileReader)
		if err != nil {
			return err
		}
	} else {
		reader = fileReader
	}
	return extractFile(reader, destDir)
}

// extractFile decompresses the tar file reader at inputFile to destDir.
func extractFile(reader io.Reader, destDir string) error {
	// Open the file for read

	// Use gzip to read from the file
	tarReader := tar.NewReader(reader)
	// Read the file sequentially
	for {
		fileHeader, err := tarReader.Next()
		switch {
		// If no more files are found, return
		case err == io.EOF:
			return nil
		// Return if hit any error
		case err != nil:
			return err
		// If next file's header is nil, just skip it.
		case fileHeader == nil:
			continue
		}
		// Create a target file locally
		localTarget := filepath.Join(destDir, fileHeader.Name)
		switch fileHeader.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(localTarget, 0755); err != nil {
				return err
			}
		// This should be tar.TypeReg, e.g regular file.
		default:
			localDir := filepath.Dir(localTarget)
			// Create a dir if it doesn't exist but it should have created dir already.
			if err := os.MkdirAll(localDir, 0755); err != nil {
				return err
			}
			localFile, err := os.Create(localTarget)
			if err != nil {
				return err
			}
			// Copy over the contents.
			if _, err = io.Copy(localFile, tarReader); err != nil {
				localFile.Close()
				return err
			}
			localFile.Close()
		}
	}
	return nil
}
