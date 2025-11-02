package github

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/dsnet/compress/bzip2"
)

// ArchiveFile represents a file from an archive
type ArchiveFile struct {
	Name    string
	Content []byte
	IsDir   bool
}

// detectArchiveType detects the archive format from URL and magic bytes
func detectArchiveType(url string, data []byte) string {
	lowerURL := strings.ToLower(url)

	// Check file extensions
	if strings.HasSuffix(lowerURL, ".tar.gz") || strings.HasSuffix(lowerURL, ".tgz") {
		return "tar.gz"
	}
	if strings.HasSuffix(lowerURL, ".tar.bz2") || strings.HasSuffix(lowerURL, ".tbz2") || strings.HasSuffix(lowerURL, ".tbz") {
		return "tar.bz2"
	}
	if strings.HasSuffix(lowerURL, ".tar") {
		return "tar"
	}
	if strings.HasSuffix(lowerURL, ".zip") {
		return "zip"
	}
	if strings.HasSuffix(lowerURL, ".gz") {
		return "gz"
	}
	if strings.HasSuffix(lowerURL, ".bz2") {
		return "bz2"
	}

	// Check magic bytes
	if len(data) >= 3 {
		// BZIP2: BZ (0x425A) followed by 'h'
		if data[0] == 0x42 && data[1] == 0x5A && data[2] == 0x68 {
			return "tar.bz2"
		}
	}
	if len(data) >= 2 {
		// ZIP: PK (0x504B)
		if data[0] == 0x50 && data[1] == 0x4B {
			return "zip"
		}
		// GZIP: 0x1f 0x8b
		if data[0] == 0x1f && data[1] == 0x8b {
			return "tar.gz"
		}
	}

	// TAR: "ustar" at offset 257
	if len(data) >= 262 && string(data[257:262]) == "ustar" {
		return "tar"
	}

	return "unknown"
}

// extractArchive extracts files based on archive type
func extractArchive(archiveType string, data []byte) ([]ArchiveFile, error) {
	switch archiveType {
	case "tar.gz", "tgz":
		return extractTarGz(data)
	case "tar.bz2", "tbz2", "tbz":
		return extractTarBz2(data)
	case "tar":
		return extractTar(bytes.NewReader(data))
	case "zip":
		return extractZip(data)
	case "gz":
		content, err := extractGzip(data)
		if err != nil {
			return nil, err
		}
		return []ArchiveFile{{Name: "file", Content: content}}, nil
	case "bz2":
		content, err := extractBzip2(data)
		if err != nil {
			return nil, err
		}
		return []ArchiveFile{{Name: "file", Content: content}}, nil
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

// extractTarGz extracts a tar.gz archive
func extractTarGz(data []byte) ([]ArchiveFile, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return extractTar(gzReader)
}

// extractTar extracts a tar archive
func extractTar(reader io.Reader) ([]ArchiveFile, error) {
	var files []ArchiveFile
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar header: %w", err)
		}

		if header.Typeflag == tar.TypeDir {
			files = append(files, ArchiveFile{
				Name:  header.Name,
				IsDir: true,
			})
			continue
		}

		// Read file content
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read tar file %s: %w", header.Name, err)
		}

		files = append(files, ArchiveFile{
			Name:    header.Name,
			Content: content,
		})
	}

	return files, nil
}

// extractZip extracts a zip archive
func extractZip(data []byte) ([]ArchiveFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zip reader: %w", err)
	}

	var files []ArchiveFile
	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			files = append(files, ArchiveFile{
				Name:  f.Name,
				IsDir: true,
			})
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip file %s: %w", f.Name, err)
		}

		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read zip file %s: %w", f.Name, err)
		}

		files = append(files, ArchiveFile{
			Name:    f.Name,
			Content: content,
		})
	}

	return files, nil
}

// extractGzip extracts a single gzipped file
func extractGzip(data []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return io.ReadAll(gzReader)
}

// extractTarBz2 extracts a tar.bz2 archive
func extractTarBz2(data []byte) ([]ArchiveFile, error) {
	bz2Reader, err := bzip2.NewReader(bytes.NewReader(data), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bzip2 reader: %w", err)
	}
	defer bz2Reader.Close()

	return extractTar(bz2Reader)
}

// extractBzip2 extracts a single bzip2 compressed file
func extractBzip2(data []byte) ([]byte, error) {
	bz2Reader, err := bzip2.NewReader(bytes.NewReader(data), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create bzip2 reader: %w", err)
	}
	defer bz2Reader.Close()

	return io.ReadAll(bz2Reader)
}
