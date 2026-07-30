package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
