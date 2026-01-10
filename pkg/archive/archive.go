/*
Copyright © 2024 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package archive provides tar.gz compression and extraction utilities for directory transfers.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Options configures archive creation behavior.
type Options struct {
	// ExcludePatterns contains glob patterns to exclude from the archive.
	ExcludePatterns []string
	// FollowSymlinks determines whether to follow symbolic links.
	FollowSymlinks bool
	// PreservePermissions determines whether to preserve file permissions.
	PreservePermissions bool
	// CompressionLevel is the gzip compression level (1-9, 0 for default).
	CompressionLevel int
}

// DefaultOptions returns default archive options.
func DefaultOptions() *Options {
	return &Options{
		ExcludePatterns:     []string{},
		FollowSymlinks:      false,
		PreservePermissions: true,
		CompressionLevel:    gzip.DefaultCompression,
	}
}

// CreateTarGz creates a tar.gz archive of the specified directory and writes it to w.
// Returns the total number of bytes written and any error.
func CreateTarGz(w io.Writer, srcPath string, opts *Options) (int64, error) {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Verify source exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return 0, fmt.Errorf("source path error: %w", err)
	}

	// Create gzip writer
	gzWriter, err := gzip.NewWriterLevel(w, opts.CompressionLevel)
	if err != nil {
		return 0, fmt.Errorf("gzip writer error: %w", err)
	}
	defer func() {
		if closeErr := gzWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("gzip close error: %w", closeErr)
		}
	}()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		if closeErr := tarWriter.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("tar close error: %w", closeErr)
		}
	}()

	// Track bytes written
	var bytesWritten int64

	// If source is a single file, just add it to the archive
	if !srcInfo.IsDir() {
		return addFileToTar(tarWriter, srcPath, filepath.Base(srcPath))
	}

	// Walk the directory tree
	err = filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for archive
		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Check if path should be excluded
		if shouldExclude(relPath, opts.ExcludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			if !opts.FollowSymlinks {
				return addSymlinkToTar(tarWriter, path, relPath)
			}
			// Follow symlink and get actual file info
			actualPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("symlink error: %w", err)
			}
			info, err = os.Stat(actualPath)
			if err != nil {
				return fmt.Errorf("symlink target error: %w", err)
			}
		}

		// Add to archive
		if info.IsDir() {
			return addDirToTar(tarWriter, relPath, info)
		}

		n, err := addFileToTar(tarWriter, path, relPath)
		bytesWritten += n
		return err
	})

	if err != nil {
		return bytesWritten, fmt.Errorf("directory walk error: %w", err)
	}

	// Close writers to flush buffers
	if err := tarWriter.Close(); err != nil {
		return bytesWritten, fmt.Errorf("tar close error: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return bytesWritten, fmt.Errorf("gzip close error: %w", err)
	}

	return bytesWritten, nil
}

// ExtractTarGz extracts a tar.gz archive from r to the destination directory.
func ExtractTarGz(r io.Reader, destPath string, opts *Options) error {
	if opts == nil {
		opts = DefaultOptions()
	}

	// Create gzip reader
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader error: %w", err)
	}
	defer func() {
		if closeErr := gzReader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("gzip reader close error: %w", closeErr)
		}
	}()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract all files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// Validate path to prevent directory traversal attacks
		if !isValidPath(header.Name) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		// Construct target path
		targetPath := filepath.Join(destPath, header.Name) // #nosec G305 - header.Name validated by isValidPath

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
			return fmt.Errorf("mkdir error: %w", err)
		}

		// Extract based on type
		switch header.Typeflag {
		case tar.TypeDir:
			if err := extractDir(targetPath, header, opts); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := extractFile(tarReader, targetPath, header, opts); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := extractSymlink(targetPath, header); err != nil {
				return err
			}
		default:
			// Skip unsupported types
			continue
		}
	}

	return nil
}

// addFileToTar adds a single file to the tar archive.
func addFileToTar(tw *tar.Writer, srcPath, archivePath string) (int64, error) {
	file, err := os.Open(srcPath) // #nosec G304 - srcPath is from filepath.Walk, validated earlier
	if err != nil {
		return 0, fmt.Errorf("open file error: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("file close error: %w", closeErr)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat file error: %w", err)
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return 0, fmt.Errorf("create header error: %w", err)
	}
	header.Name = filepath.ToSlash(archivePath)

	if err := tw.WriteHeader(header); err != nil {
		return 0, fmt.Errorf("write header error: %w", err)
	}

	n, err := io.Copy(tw, file)
	if err != nil {
		return n, fmt.Errorf("copy file error: %w", err)
	}

	return n, nil
}

// addDirToTar adds a directory entry to the tar archive.
func addDirToTar(tw *tar.Writer, archivePath string, info os.FileInfo) error {
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("create dir header error: %w", err)
	}
	header.Name = filepath.ToSlash(archivePath) + "/"

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write dir header error: %w", err)
	}

	return nil
}

// addSymlinkToTar adds a symbolic link to the tar archive.
func addSymlinkToTar(tw *tar.Writer, srcPath, archivePath string) error {
	linkTarget, err := os.Readlink(srcPath)
	if err != nil {
		return fmt.Errorf("readlink error: %w", err)
	}

	info, err := os.Lstat(srcPath)
	if err != nil {
		return fmt.Errorf("lstat error: %w", err)
	}

	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return fmt.Errorf("create symlink header error: %w", err)
	}
	header.Name = filepath.ToSlash(archivePath)

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("write symlink header error: %w", err)
	}

	return nil
}

// extractDir creates a directory with proper permissions.
func extractDir(targetPath string, header *tar.Header, opts *Options) error {
	mode := os.FileMode(0750)
	if opts.PreservePermissions {
		mode = header.FileInfo().Mode()
	}

	if err := os.MkdirAll(targetPath, mode); err != nil {
		return fmt.Errorf("create dir error: %w", err)
	}

	return nil
}

// extractFile extracts a regular file from the archive.
func extractFile(r io.Reader, targetPath string, header *tar.Header, opts *Options) error {
	mode := os.FileMode(0600)
	if opts.PreservePermissions {
		mode = header.FileInfo().Mode()
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode) // #nosec G304 - targetPath validated by isValidPath
	if err != nil {
		return fmt.Errorf("create file error: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("file close error: %w", closeErr)
		}
	}()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("write file error: %w", err)
	}

	if opts.PreservePermissions {
		if err := os.Chmod(targetPath, header.FileInfo().Mode().Perm()); err != nil {
			// Non-fatal, just log
			_ = err
		}
	}

	// Preserve modification time
	if opts.PreservePermissions {
		if err := os.Chtimes(targetPath, header.AccessTime, header.ModTime); err != nil {
			// Non-fatal, just log
			_ = err
		}
	}

	return nil
}

// extractSymlink creates a symbolic link.
func extractSymlink(targetPath string, header *tar.Header) error {
	// Remove if exists (ignore error if file doesn't exist)
	_ = os.Remove(targetPath)

	if err := os.Symlink(header.Linkname, targetPath); err != nil {
		return fmt.Errorf("create symlink error: %w", err)
	}

	return nil
}

// shouldExclude checks if a path matches any exclude pattern.
func shouldExclude(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
		// Also check if full path matches
		matched, err = filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// isValidPath validates that a path doesn't contain directory traversal sequences.
func isValidPath(path string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" {
		return false
	}
	if strings.HasPrefix(cleanPath, "/") {
		return false
	}
	for _, part := range strings.Split(cleanPath, "/") {
		if part == ".." {
			return false
		}
	}
	if strings.Contains(path, "..") {
		return false
	}
	return true
}

// IsDirectory checks if the given path is a directory.
func IsDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
