package backup

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportDataDir writes the contents of dataDir into w as a zip archive.
func ExportDataDir(dataDir string, w io.Writer) (err error) {
	zw := zip.NewWriter(w)
	defer func() {
		cerr := zw.Close()
		if err == nil {
			err = cerr
		}
	}()

	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		// Skip symbolic links to avoid backing up unintended external paths.
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		// Normalize to forward slashes for zip.
		zipName := filepath.ToSlash(rel)
		if info.IsDir() {
			return nil
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = zipName
		h.Method = zip.Deflate
		fw, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(fw, f)
		return err
	})
	return err
}

// BackupFilename returns a timestamped backup filename.
func BackupFilename(prefix string) string {
	return fmt.Sprintf("%s_%s.zip", prefix, time.Now().Format("20060102_150405"))
}

// ImportDataDir restores dataDir from a zip archive read from r.
// It returns the path of the automatic backup copy of the previous dataDir.
func ImportDataDir(dataDir string, r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}

	if err := validateBackupZip(zr); err != nil {
		return "", err
	}

	bak := dataDir + ".bak." + time.Now().Format("20060102_150405")
	if err := copyDir(dataDir, bak); err != nil {
		return "", fmt.Errorf("failed to backup current data dir: %w", err)
	}

	if err := os.RemoveAll(dataDir); err != nil {
		restoreDir(bak, dataDir)
		return "", fmt.Errorf("failed to clear data dir: %w", err)
	}

	if err := extractZip(zr, dataDir); err != nil {
		restoreDir(bak, dataDir)
		return "", fmt.Errorf("failed to extract backup: %w", err)
	}

	return bak, nil
}

func validateBackupZip(zr *zip.Reader) error {
	for _, f := range zr.File {
		if f.Name == "database.db" {
			return nil
		}
	}
	return fmt.Errorf("backup archive missing database.db")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		sf, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sf.Close()
		df, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer df.Close()
		_, err = io.Copy(df, sf)
		return err
	})
}

func restoreDir(src, dst string) {
	_ = os.RemoveAll(dst)
	_ = copyDir(src, dst)
}

func extractZip(zr *zip.Reader, dst string) error {
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		target := filepath.Join(dstAbs, filepath.FromSlash(f.Name))
		target = filepath.Clean(target)
		if !strings.HasPrefix(target, dstAbs+string(os.PathSeparator)) && target != dstAbs {
			return fmt.Errorf("invalid zip entry %q: escapes destination", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		tf, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(tf, rc)
		rc.Close()
		tf.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
