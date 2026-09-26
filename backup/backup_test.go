package backup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateValidateAndStageBackup(t *testing.T) {
	root := t.TempDir()
	db := filepath.Join(root, "gotify.db")
	images := filepath.Join(root, "images")
	plugins := filepath.Join(root, "plugins")
	if err := os.WriteFile(db, []byte("database"), 0o600); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(images, 0o700); err != nil { t.Fatal(err) }
	if err := os.MkdirAll(plugins, 0o700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(images, "logo.png"), []byte("image"), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(plugins, "test.so"), []byte("plugin"), 0o600); err != nil { t.Fatal(err) }

	archive := filepath.Join(root, "backup.zip")
	if err := CreateArchive(archive, db, images, plugins, "0.5.0", "abc123"); err != nil { t.Fatal(err) }
	manifest, err := ValidateArchive(archive)
	if err != nil { t.Fatal(err) }
	if manifest.Version != "0.5.0" || manifest.Commit != "abc123" {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}

	input, err := os.Open(archive)
	if err != nil { t.Fatal(err) }
	defer input.Close()
	staged := filepath.Join(root, "staged.zip")
	stagedManifest, err := StageRestore(input, staged)
	if err != nil { t.Fatal(err) }
	if stagedManifest.Version != "0.5.0" { t.Fatalf("unexpected staged manifest: %#v", stagedManifest) }
}

func TestValidateArchiveRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "unsafe.zip")
	file, err := os.Create(path)
	if err != nil { t.Fatal(err) }
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../escape")
	if err != nil { t.Fatal(err) }
	_, _ = entry.Write([]byte("bad"))
	if err := writer.Close(); err != nil { t.Fatal(err) }
	if err := file.Close(); err != nil { t.Fatal(err) }
	if _, err := ValidateArchive(path); err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Fatalf("expected unsafe path error, got %v", err)
	}
}

func TestApplyPendingRestore(t *testing.T) {
	root := t.TempDir()
	currentDB := filepath.Join(root, "gotify.db")
	images := filepath.Join(root, "images")
	plugins := filepath.Join(root, "plugins")
	if err := os.WriteFile(currentDB, []byte("old"), 0o600); err != nil { t.Fatal(err) }

	sourceRoot := t.TempDir()
	sourceDB := filepath.Join(sourceRoot, "gotify.db")
	sourceImages := filepath.Join(sourceRoot, "images")
	sourcePlugins := filepath.Join(sourceRoot, "plugins")
	_ = os.MkdirAll(sourceImages, 0o700)
	_ = os.MkdirAll(sourcePlugins, 0o700)
	_ = os.WriteFile(sourceDB, []byte("new"), 0o600)
	_ = os.WriteFile(filepath.Join(sourceImages, "a.txt"), []byte("image"), 0o600)
	_ = os.WriteFile(filepath.Join(sourcePlugins, "p.so"), []byte("plugin"), 0o600)
	if err := CreateArchive(PendingPath(currentDB), sourceDB, sourceImages, sourcePlugins, "0.5.0", "abc"); err != nil { t.Fatal(err) }

	applied, err := ApplyPending(currentDB, images, plugins)
	if err != nil { t.Fatal(err) }
	if !applied { t.Fatal("expected pending restore to be applied") }
	data, err := os.ReadFile(currentDB)
	if err != nil { t.Fatal(err) }
	if string(data) != "new" { t.Fatalf("expected restored database, got %q", data) }
}
