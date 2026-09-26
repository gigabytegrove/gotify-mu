package operations

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
	ManifestName = "gotify-mu-backup.json"
	PendingRestoreName = ".gotify-mu-restore-pending.zip"
)

func DataDirectory(dialect, connection string) string {
	if dialect != "sqlite3" && dialect != "sqlite" {
		return ""
	}
	candidate := strings.TrimPrefix(strings.TrimSpace(connection), "file:")
	if query := strings.IndexByte(candidate, '?'); query >= 0 {
		candidate = candidate[:query]
	}
	if candidate == "" || candidate == ":memory:" {
		return ""
	}
	return filepath.Clean(filepath.Dir(candidate))
}

func DatabaseFile(dialect, connection string) string {
	if dialect != "sqlite3" && dialect != "sqlite" {
		return ""
	}
	candidate := strings.TrimPrefix(strings.TrimSpace(connection), "file:")
	if query := strings.IndexByte(candidate, '?'); query >= 0 {
		candidate = candidate[:query]
	}
	if candidate == "" || candidate == ":memory:" {
		return ""
	}
	return filepath.Clean(candidate)
}

type BackupManifest struct {
	FormatVersion int       `json:"formatVersion"`
	Product       string    `json:"product"`
	Version       string    `json:"version"`
	Commit        string    `json:"commit"`
	CreatedAt     time.Time `json:"createdAt"`
	DatabasePath  string    `json:"databasePath"`
}

func CreateBackupBundle(dataDir, databasePath, snapshotPath, destination, version, commit string) error {
	dataDir = filepath.Clean(dataDir)
	databasePath = filepath.Clean(databasePath)
	databaseRel, err := filepath.Rel(dataDir, databasePath)
	if err != nil || strings.HasPrefix(databaseRel, "..") {
		return errors.New("database must be inside the Gotify MU data directory")
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	archive := zip.NewWriter(out)
	closeWithError := func(primary error) error {
		zipErr := archive.Close()
		fileErr := out.Close()
		if primary != nil { return primary }
		if zipErr != nil { return zipErr }
		return fileErr
	}

	manifest := BackupManifest{
		FormatVersion: 1,
		Product: "Gotify MU",
		Version: version,
		Commit: commit,
		CreatedAt: time.Now().UTC(),
		DatabasePath: filepath.ToSlash(databaseRel),
	}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil { return closeWithError(err) }
	header, err := archive.Create(ManifestName)
	if err != nil { return closeWithError(err) }
	if _, err := header.Write(body); err != nil { return closeWithError(err) }

	if err := addFileToZip(archive, snapshotPath, filepath.ToSlash(databaseRel)); err != nil {
		return closeWithError(err)
	}

	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil { return walkErr }
		if path == dataDir { return nil }
		rel, relErr := filepath.Rel(dataDir, path)
		if relErr != nil { return relErr }
		relSlash := filepath.ToSlash(rel)
		if relSlash == filepath.ToSlash(databaseRel) ||
			relSlash == PendingRestoreName ||
			strings.HasPrefix(relSlash, "backups/") ||
			strings.HasPrefix(relSlash, ".restore-work-") {
			if info.IsDir() && (strings.HasPrefix(relSlash, "backups") || strings.HasPrefix(relSlash, ".restore-work-")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() { return nil }
		return addFileToZip(archive, path, relSlash)
	})
	return closeWithError(err)
}

func addFileToZip(archive *zip.Writer, source, name string) error {
	info, err := os.Stat(source)
	if err != nil { return err }
	header, err := zip.FileInfoHeader(info)
	if err != nil { return err }
	header.Name = filepath.ToSlash(name)
	header.Method = zip.Deflate
	writer, err := archive.CreateHeader(header)
	if err != nil { return err }
	file, err := os.Open(source)
	if err != nil { return err }
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}

func ValidateAndStageRestore(source io.Reader, dataDir string, maxBytes int64) (BackupManifest, error) {
	dataDir = filepath.Clean(dataDir)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return BackupManifest{}, err
	}
	temp, err := os.CreateTemp(dataDir, ".restore-upload-*.zip")
	if err != nil { return BackupManifest{}, err }
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	written, err := io.Copy(temp, io.LimitReader(source, maxBytes+1))
	closeErr := temp.Close()
	if err != nil { return BackupManifest{}, err }
	if closeErr != nil { return BackupManifest{}, closeErr }
	if written > maxBytes { return BackupManifest{}, errors.New("backup exceeds restore upload limit") }

	manifest, err := ValidateBackupBundle(tempPath)
	if err != nil { return BackupManifest{}, err }
	pending := filepath.Join(dataDir, PendingRestoreName)
	if err := os.Rename(tempPath, pending); err != nil { return BackupManifest{}, err }
	return manifest, nil
}

func ValidateBackupBundle(path string) (BackupManifest, error) {
	reader, err := zip.OpenReader(path)
	if err != nil { return BackupManifest{}, err }
	defer reader.Close()
	var manifest BackupManifest
	foundManifest := false
	foundDatabase := false
	for _, entry := range reader.File {
		clean := filepath.Clean(entry.Name)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
			return manifest, fmt.Errorf("backup contains invalid path %q", entry.Name)
		}
		if filepath.ToSlash(clean) == ManifestName {
			file, openErr := entry.Open()
			if openErr != nil { return manifest, openErr }
			decodeErr := json.NewDecoder(io.LimitReader(file, 1<<20)).Decode(&manifest)
			_ = file.Close()
			if decodeErr != nil { return manifest, decodeErr }
			foundManifest = true
		}
	}
	if !foundManifest || manifest.Product != "Gotify MU" || manifest.FormatVersion != 1 {
		return manifest, errors.New("file is not a supported Gotify MU backup")
	}
	for _, entry := range reader.File {
		if filepath.ToSlash(filepath.Clean(entry.Name)) == filepath.ToSlash(manifest.DatabasePath) {
			foundDatabase = true
			break
		}
	}
	if !foundDatabase {
		return manifest, errors.New("backup database is missing")
	}
	return manifest, nil
}

func ApplyPendingRestore(dataDir string) (string, bool, error) {
	dataDir = filepath.Clean(dataDir)
	pending := filepath.Join(dataDir, PendingRestoreName)
	if _, err := os.Stat(pending); errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	if _, err := ValidateBackupBundle(pending); err != nil {
		return "", true, err
	}

	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", true, err
	}
	safetyPath := filepath.Join(backupDir, "pre-restore-"+time.Now().UTC().Format("20060102-150405")+".zip")
	if err := createColdSafetyBundle(dataDir, safetyPath); err != nil {
		return "", true, fmt.Errorf("create pre-restore safety backup: %w", err)
	}

	workDir, err := os.MkdirTemp(dataDir, ".restore-work-*")
	if err != nil { return safetyPath, true, err }
	defer os.RemoveAll(workDir)
	if err := extractBackup(pending, workDir); err != nil {
		return safetyPath, true, err
	}

	entries, err := os.ReadDir(workDir)
	if err != nil { return safetyPath, true, err }
	for _, entry := range entries {
		if entry.Name() == ManifestName { continue }
		source := filepath.Join(workDir, entry.Name())
		target := filepath.Join(dataDir, entry.Name())
		if err := os.RemoveAll(target); err != nil { return safetyPath, true, err }
		if err := os.Rename(source, target); err != nil {
			return safetyPath, true, err
		}
	}
	if err := os.Remove(pending); err != nil {
		return safetyPath, true, err
	}
	return safetyPath, true, nil
}

func createColdSafetyBundle(dataDir, destination string) error {
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil { return err }
	archive := zip.NewWriter(out)
	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil { return walkErr }
		if path == dataDir { return nil }
		rel, relErr := filepath.Rel(dataDir, path)
		if relErr != nil { return relErr }
		relSlash := filepath.ToSlash(rel)
		if relSlash == PendingRestoreName || strings.HasPrefix(relSlash, "backups/") || strings.HasPrefix(relSlash, ".restore-work-") {
			if info.IsDir() { return filepath.SkipDir }
			return nil
		}
		if !info.Mode().IsRegular() { return nil }
		return addFileToZip(archive, path, relSlash)
	})
	zipErr := archive.Close()
	fileErr := out.Close()
	if err != nil { return err }
	if zipErr != nil { return zipErr }
	return fileErr
}

func extractBackup(path, destination string) error {
	reader, err := zip.OpenReader(path)
	if err != nil { return err }
	defer reader.Close()
	root := filepath.Clean(destination) + string(os.PathSeparator)
	for _, entry := range reader.File {
		if filepath.ToSlash(filepath.Clean(entry.Name)) == ManifestName { continue }
		target := filepath.Join(destination, entry.Name)
		clean := filepath.Clean(target)
		if !strings.HasPrefix(clean+string(os.PathSeparator), root) {
			return fmt.Errorf("backup contains invalid path %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(clean, 0o700); err != nil { return err }
			continue
		}
		if err := os.MkdirAll(filepath.Dir(clean), 0o700); err != nil { return err }
		source, err := entry.Open()
		if err != nil { return err }
		targetFile, err := os.OpenFile(clean, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode())
		if err != nil { _=source.Close(); return err }
		_, copyErr := io.Copy(targetFile, source)
		closeErr := targetFile.Close()
		_ = source.Close()
		if copyErr != nil { return copyErr }
		if closeErr != nil { return closeErr }
	}
	return nil
}
