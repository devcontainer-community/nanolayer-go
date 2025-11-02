package github

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"strings"
	"testing"

	bzip2 "github.com/dsnet/compress/bzip2"
)

type archiveEntry struct {
	name  string
	body  []byte
	isDir bool
}

func TestDetectArchiveType(t *testing.T) {
	tarEntries := []archiveEntry{{name: "dir/", isDir: true}, {name: "dir/file.txt", body: []byte("hello")}}
	tarData := createTarArchive(t, tarEntries)

	tests := []struct {
		name string
		url  string
		data []byte
		want string
	}{
		{name: "tar gz extension", url: "https://example.com/foo.tar.gz", data: []byte{}, want: "tar.gz"},
		{name: "tgz extension", url: "foo.tgz", data: []byte{}, want: "tar.gz"},
		{name: "tar bz2 extension", url: "foo.tbz2", data: []byte{}, want: "tar.bz2"},
		{name: "tbz extension", url: "foo.tbz", data: []byte{}, want: "tar.bz2"},
		{name: "tar extension", url: "foo.tar", data: []byte{}, want: "tar"},
		{name: "zip extension", url: "foo.zip", data: []byte{}, want: "zip"},
		{name: "gz extension", url: "foo.gz", data: []byte{}, want: "gz"},
		{name: "bz2 extension", url: "foo.bz2", data: []byte{}, want: "bz2"},
		{name: "zip magic", url: "foo.bin", data: createZipArchive(t, tarEntries), want: "zip"},
		{name: "gzip magic", url: "foo.bin", data: []byte{0x1f, 0x8b}, want: "tar.gz"},
		{name: "bzip magic", url: "foo.bin", data: []byte{0x42, 0x5A, 0x68}, want: "tar.bz2"},
		{name: "tar magic", url: "foo.bin", data: tarData, want: "tar"},
		{name: "unknown", url: "foo.bin", data: []byte{0x00}, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectArchiveType(tt.url, tt.data)
			if got != tt.want {
				t.Fatalf("detectArchiveType(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestExtractArchive(t *testing.T) {
	baseEntries := []archiveEntry{
		{name: "dir/", isDir: true},
		{name: "dir/file.txt", body: []byte("hello")},
		{name: "root.bin", body: []byte{0x00, 0x01, 0x02}},
	}

	tarData := createTarArchive(t, baseEntries)
	tarGzData := compressGzipData(t, tarData)
	tarBz2Data := compressBzip2Data(t, tarData)
	zipData := createZipArchive(t, baseEntries)
	gzSingle := compressGzipData(t, []byte("solo"))
	bz2Single := compressBzip2Data(t, []byte("solo-bz"))

	tests := []struct {
		name        string
		archiveType string
		data        []byte
		want        []archiveEntry
	}{
		{name: "tar.gz", archiveType: "tar.gz", data: tarGzData, want: baseEntries},
		{name: "tar.bz2", archiveType: "tar.bz2", data: tarBz2Data, want: baseEntries},
		{name: "tar", archiveType: "tar", data: tarData, want: baseEntries},
		{name: "zip", archiveType: "zip", data: zipData, want: baseEntries},
		{name: "gz", archiveType: "gz", data: gzSingle, want: []archiveEntry{{name: "file", body: []byte("solo")}}},
		{name: "bz2", archiveType: "bz2", data: bz2Single, want: []archiveEntry{{name: "file", body: []byte("solo-bz")}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractArchive(tt.archiveType, tt.data)
			if err != nil {
				t.Fatalf("extractArchive returned error: %v", err)
			}
			assertArchiveFiles(t, got, tt.want)
		})
	}
}

func TestExtractArchiveUnsupported(t *testing.T) {
	if _, err := extractArchive("rar", []byte("data")); err == nil {
		t.Fatalf("expected error for unsupported archive type")
	}
}

func TestExtractTarGzInvalid(t *testing.T) {
	if _, err := extractTarGz([]byte("not-a-gzip")); err == nil {
		t.Fatalf("expected error for invalid gzip data")
	}
}

func TestExtractTarBz2Invalid(t *testing.T) {
	if _, err := extractTarBz2([]byte("not-a-bzip")); err == nil {
		t.Fatalf("expected error for invalid bzip2 data")
	}
}

func TestExtractArchiveInvalidGzip(t *testing.T) {
	if _, err := extractArchive("gz", []byte("bad")); err == nil {
		t.Fatalf("expected error for invalid gzip data")
	}
}

func createTarArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for _, entry := range entries {
		hdr := &tar.Header{
			Name: entry.name,
			Mode: 0o644,
			Size: int64(len(entry.body)),
		}

		if entry.isDir {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0o755
			hdr.Size = 0
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("failed to write tar header: %v", err)
		}

		if entry.isDir {
			continue
		}

		if _, err := tw.Write(entry.body); err != nil {
			t.Fatalf("failed to write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	return buf.Bytes()
}

func createZipArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, entry := range entries {
		if entry.isDir {
			name := entry.name
			if !strings.HasSuffix(name, "/") {
				name += "/"
			}
			hdr := &zip.FileHeader{Name: name, Method: zip.Store}
			hdr.SetMode(0o755 | os.ModeDir)
			if _, err := zw.CreateHeader(hdr); err != nil {
				t.Fatalf("failed to write zip dir header: %v", err)
			}
			continue
		}

		writer, err := zw.Create(entry.name)
		if err != nil {
			t.Fatalf("failed to create zip entry: %v", err)
		}

		if _, err := writer.Write(entry.body); err != nil {
			t.Fatalf("failed to write zip content: %v", err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return buf.Bytes()
}

func compressGzipData(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(data); err != nil {
		t.Fatalf("failed to write gzip data: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}
	return buf.Bytes()
}

func compressBzip2Data(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	bw, err := bzip2.NewWriter(&buf, nil)
	if err != nil {
		t.Fatalf("failed to create bzip2 writer: %v", err)
	}
	if _, err := bw.Write(data); err != nil {
		t.Fatalf("failed to write bzip2 data: %v", err)
	}
	if err := bw.Close(); err != nil {
		t.Fatalf("failed to close bzip2 writer: %v", err)
	}
	return buf.Bytes()
}

func assertArchiveFiles(t *testing.T, got []ArchiveFile, want []archiveEntry) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i].Name != want[i].name {
			t.Fatalf("file %d name = %q, want %q", i, got[i].Name, want[i].name)
		}
		if got[i].IsDir != want[i].isDir {
			t.Fatalf("file %q IsDir = %v, want %v", got[i].Name, got[i].IsDir, want[i].isDir)
		}
		if want[i].isDir {
			continue
		}
		if !bytes.Equal(got[i].Content, want[i].body) {
			t.Fatalf("file %q content mismatch", got[i].Name)
		}
	}
}
