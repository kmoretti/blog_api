package backup

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExportDataDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("dbdata"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(tmp, "config")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "system_config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := ExportDataDir(tmp, &buf); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.File) != 2 {
		t.Fatalf("expected 2 files, got %d", len(r.File))
	}

	want := map[string]string{
		"database.db":               "dbdata",
		"config/system_config.json": "{}",
	}
	got := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		got[f.Name] = string(data)
	}
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s: expected %q, got %q", name, content, got[name])
		}
	}
}

func TestImportDataDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build a backup zip in memory.
	var buf bytes.Buffer
	if err := ExportDataDir(tmp, &buf); err != nil {
		t.Fatal(err)
	}

	// Mutate the original dir.
	if err := os.WriteFile(filepath.Join(tmp, "database.db"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	bak, err := ImportDataDir(tmp, &buf)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "database.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("expected restored content 'old', got %q", string(data))
	}
	if _, err := os.Stat(bak); err != nil {
		t.Fatalf("backup dir missing: %v", err)
	}
}

func TestValidateBackupZip_MissingDatabase(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("other.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	if err := validateBackupZip(zr); err == nil {
		t.Fatal("expected error for archive missing database.db")
	}
}

func TestExtractZip_ZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.Create("../evil.txt"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	if err := extractZip(zr, dst); err == nil {
		t.Fatal("expected zip-slip entry to be rejected")
	}

	evil := filepath.Clean(filepath.Join(dst, "..", "evil.txt"))
	if _, err := os.Stat(evil); !os.IsNotExist(err) {
		t.Fatalf("zip-slip file was created outside destination: %v", err)
	}
}

func TestExportModuleSystemConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "system_config.json"), []byte(`{"site":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	env, err := ExportModule(nil, "system_config", tmp)
	if err != nil {
		t.Fatal(err)
	}
	if env.Module != "system_config" {
		t.Fatalf("expected module system_config, got %s", env.Module)
	}
}
