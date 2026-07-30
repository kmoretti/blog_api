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
		"database.db":             "dbdata",
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
