package backup

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	FormatVersion = 1
	MaxRestoreBytes int64 = 2 << 30
)

type Manifest struct {
	FormatVersion int       `json:"formatVersion"`
	CreatedAt     time.Time `json:"createdAt"`
	Version       string    `json:"version"`
	Commit        string    `json:"commit"`
}

func CreateArchive(destination, databaseSnapshot, imagesDir, pluginsDir, version, commit string) error {
	out, err := os.Create(destination)
	if err != nil { return err }
	defer out.Close()
	writer := zip.NewWriter(out)
	defer writer.Close()

	manifest, err := json.MarshalIndent(Manifest{
		FormatVersion: FormatVersion,
		CreatedAt: time.Now().UTC(),
		Version: version,
		Commit: commit,
	}, "", "  ")
	if err != nil { return err }
	if err := writeBytes(writer, "manifest.json", manifest, 0o600); err != nil { return err }
	if err := addFile(writer, databaseSnapshot, "database/gotify.db"); err != nil { return err }
	if err := addTree(writer, imagesDir, "files/images"); err != nil { return err }
	if err := addTree(writer, pluginsDir, "files/plugins"); err != nil { return err }
	return writer.Close()
}

func StageRestore(source io.Reader, destination string) (*Manifest, error) {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil { return nil, err }
	temp, err := os.CreateTemp(filepath.Dir(destination), ".gotify-mu-restore-*.zip")
	if err != nil { return nil, err }
	tempPath := temp.Name()
	defer func(){ _ = os.Remove(tempPath) }()

	written, err := io.Copy(temp, io.LimitReader(source, MaxRestoreBytes+1))
	closeErr := temp.Close()
	if err != nil { return nil, err }
	if closeErr != nil { return nil, closeErr }
	if written > MaxRestoreBytes { return nil, errors.New("backup exceeds the 2 GiB restore limit") }

	manifest, err := ValidateArchive(tempPath)
	if err != nil { return nil, err }
	if err := os.Chmod(tempPath, 0o600); err != nil { return nil, err }
	if err := os.Rename(tempPath, destination); err != nil { return nil, err }
	return manifest, nil
}

func ValidateArchive(path string) (*Manifest, error) {
	reader, err := zip.OpenReader(path)
	if err != nil { return nil, fmt.Errorf("open backup: %w", err) }
	defer reader.Close()

	var manifest *Manifest
	hasDatabase := false
	for _, file := range reader.File {
		name, err := safeArchiveName(file.Name)
		if err != nil { return nil, err }
		switch name {
		case "manifest.json":
			if file.UncompressedSize64 > 1<<20 { return nil, errors.New("backup manifest is too large") }
			source, err := file.Open()
			if err != nil { return nil, err }
			data, readErr := io.ReadAll(io.LimitReader(source, 1<<20))
			source.Close()
			if readErr != nil { return nil, readErr }
			var parsed Manifest
			if err := json.Unmarshal(data, &parsed); err != nil { return nil, errors.New("backup manifest is invalid") }
			if parsed.FormatVersion != FormatVersion {
				return nil, fmt.Errorf("unsupported backup format version %d", parsed.FormatVersion)
			}
			manifest = &parsed
		case "database/gotify.db":
			hasDatabase = true
		}
	}
	if manifest == nil { return nil, errors.New("backup manifest is missing") }
	if !hasDatabase { return nil, errors.New("backup database is missing") }
	return manifest, nil
}

func PendingPath(databasePath string) string {
	return filepath.Join(filepath.Dir(databasePath), ".gotify-mu-restore.zip")
}

func ApplyPending(databasePath, imagesDir, pluginsDir string) (bool, error) {
	pending := PendingPath(databasePath)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) { return false, nil } else if err != nil { return false, err }
	if _, err := ValidateArchive(pending); err != nil { return false, err }

	work, err := os.MkdirTemp(filepath.Dir(databasePath), ".gotify-mu-restore-work-*")
	if err != nil { return false, err }
	defer os.RemoveAll(work)
	if err := extractArchive(pending, work); err != nil { return false, err }

	stamp := time.Now().UTC().Format("20060102-150405")
	emergency := filepath.Join(filepath.Dir(databasePath), "pre-restore-"+stamp)
	if err := os.MkdirAll(emergency, 0o700); err != nil { return false, err }
	if err := copyExisting(databasePath, filepath.Join(emergency, "gotify.db")); err != nil { return false, err }
	if err := copyTreeIfExists(imagesDir, filepath.Join(emergency, "images")); err != nil { return false, err }
	if err := copyTreeIfExists(pluginsDir, filepath.Join(emergency, "plugins")); err != nil { return false, err }

	restoreDB := filepath.Join(work, "database", "gotify.db")
	if err := replaceFile(restoreDB, databasePath, 0o600); err != nil { return false, err }
	if err := replaceTree(filepath.Join(work, "files", "images"), imagesDir); err != nil { return false, err }
	if err := replaceTree(filepath.Join(work, "files", "plugins"), pluginsDir); err != nil { return false, err }

	if err := os.Remove(pending); err != nil { return false, err }
	return true, nil
}

func writeBytes(writer *zip.Writer, name string, data []byte, mode os.FileMode) error {
	header := &zip.FileHeader{Name:name, Method:zip.Deflate}
	header.SetMode(mode)
	header.SetModTime(time.Now())
	target, err := writer.CreateHeader(header)
	if err != nil { return err }
	_, err = target.Write(data)
	return err
}

func addFile(writer *zip.Writer, source, name string) error {
	info, err := os.Stat(source)
	if err != nil { return err }
	header, err := zip.FileInfoHeader(info)
	if err != nil { return err }
	header.Name = name
	header.Method = zip.Deflate
	target, err := writer.CreateHeader(header)
	if err != nil { return err }
	input, err := os.Open(source)
	if err != nil { return err }
	defer input.Close()
	_, err = io.Copy(target, input)
	return err
}

func addTree(writer *zip.Writer, root, prefix string) error {
	root = filepath.Clean(root)
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) { return nil } else if err != nil { return err }
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if info.IsDir() { return nil }
		rel, err := filepath.Rel(root, path)
		if err != nil { return err }
		return addFile(writer, path, filepath.ToSlash(filepath.Join(prefix, rel)))
	})
}

func safeArchiveName(raw string) (string, error) {
	name := filepath.ToSlash(filepath.Clean(raw))
	if name == "." || strings.HasPrefix(name, "../") || strings.HasPrefix(name, "/") || strings.Contains(name, ":") {
		return "", fmt.Errorf("backup contains unsafe path %q", raw)
	}
	if name != "manifest.json" && name != "database/gotify.db" &&
		!strings.HasPrefix(name, "files/images/") && !strings.HasPrefix(name, "files/plugins/") {
		return "", fmt.Errorf("backup contains unsupported path %q", raw)
	}
	return name, nil
}

func extractArchive(path, destination string) error {
	reader, err := zip.OpenReader(path)
	if err != nil { return err }
	defer reader.Close()
	for _, file := range reader.File {
		name, err := safeArchiveName(file.Name)
		if err != nil { return err }
		if file.FileInfo().IsDir() { continue }
		target := filepath.Join(destination, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil { return err }
		source, err := file.Open()
		if err != nil { return err }
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil { source.Close(); return err }
		_, copyErr := io.Copy(out, io.LimitReader(source, MaxRestoreBytes))
		closeErr := out.Close()
		source.Close()
		if copyErr != nil { return copyErr }
		if closeErr != nil { return closeErr }
	}
	return nil
}

func replaceFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil { return err }
	temp := destination + ".restore-new"
	if err := copyFile(source, temp, mode); err != nil { return err }
	return os.Rename(temp, destination)
}

func replaceTree(source, destination string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil { return err }
	temp := strings.TrimRight(destination, string(filepath.Separator)) + ".restore-new"
	_ = os.RemoveAll(temp)
	if err := copyTreeIfExists(source, temp); err != nil { return err }
	old := strings.TrimRight(destination, string(filepath.Separator)) + ".restore-old"
	_ = os.RemoveAll(old)
	if _, err := os.Stat(destination); err == nil {
		if err := os.Rename(destination, old); err != nil { return err }
	}
	if err := os.Rename(temp, destination); err != nil {
		if _, oldErr := os.Stat(old); oldErr == nil { _ = os.Rename(old, destination) }
		return err
	}
	_ = os.RemoveAll(old)
	return nil
}

func copyExisting(source, destination string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) { return nil } else if err != nil { return err }
	return copyFile(source, destination, 0o600)
}

func copyFile(source, destination string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil { return err }
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil { return err }
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil { return err }
	_, copyErr := io.Copy(out, input)
	closeErr := out.Close()
	if copyErr != nil { return copyErr }
	return closeErr
}

func copyTreeIfExists(source, destination string) error {
	if _, err := os.Stat(source); errors.Is(err, os.ErrNotExist) { return nil } else if err != nil { return err }
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		rel, err := filepath.Rel(source, path)
		if err != nil { return err }
		target := filepath.Join(destination, rel)
		if info.IsDir() { return os.MkdirAll(target, 0o700) }
		return copyFile(path, target, info.Mode().Perm())
	})
}
