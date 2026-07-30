package backup

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blog_api/src/model"
	"gorm.io/gorm"
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
			return nil
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
		if rerr := restoreDir(bak, dataDir); rerr != nil {
			err = errors.Join(err, rerr)
		}
		return "", fmt.Errorf("failed to clear data dir: %w", err)
	}

	if err := extractZip(zr, dataDir); err != nil {
		if rerr := restoreDir(bak, dataDir); rerr != nil {
			err = errors.Join(err, rerr)
		}
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
		if info.Mode()&os.ModeSymlink != 0 {
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
		_, copyErr := io.Copy(df, sf)
		closeErr := df.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func restoreDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("failed to remove %q: %w", dst, err)
	}
	if err := copyDir(src, dst); err != nil {
		return fmt.Errorf("failed to copy %q to %q: %w", src, dst, err)
	}
	return nil
}

func extractZip(zr *zip.Reader, dst string) error {
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		target := filepath.Join(dstAbs, filepath.FromSlash(f.Name))
		target = filepath.Clean(target)
		rel, err := filepath.Rel(dstAbs, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
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

// ExportModule exports a single module as an ExportEnvelope.
func ExportModule(db *gorm.DB, module string, dataDir string) (*model.ExportEnvelope, error) {
	switch module {
	case "system_config":
		return exportSystemConfig(dataDir)
	case "friend_links":
		return exportTable(db, module, &model.FriendWebsite{})
	case "moments":
		return exportMoments(db)
	case "friend_rss":
		return exportFriendRss(db)
	case "images":
		return exportTable(db, module, &model.Image{})
	default:
		return nil, fmt.Errorf("unknown module: %s", module)
	}
}

func exportSystemConfig(dataDir string) (*model.ExportEnvelope, error) {
	path := filepath.Join(dataDir, "config", "system_config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "system_config",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      1,
		Items:      cfg,
	}, nil
}

func exportTable(db *gorm.DB, module string, tableModel interface{}) (*model.ExportEnvelope, error) {
	var items []map[string]interface{}
	if err := db.Model(tableModel).Find(&items).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     module,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(items),
		Items:      items,
	}, nil
}

func exportMoments(db *gorm.DB) (*model.ExportEnvelope, error) {
	var moments []model.Moment
	if err := db.Find(&moments).Error; err != nil {
		return nil, err
	}
	var media []model.MomentMedia
	if err := db.Find(&media).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "moments",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(moments),
		Items: map[string]interface{}{
			"moments":       moments,
			"moments_media": media,
		},
	}, nil
}

func exportFriendRss(db *gorm.DB) (*model.ExportEnvelope, error) {
	var feeds []model.FriendRss
	if err := db.Find(&feeds).Error; err != nil {
		return nil, err
	}
	var posts []model.RssPost
	if err := db.Find(&posts).Error; err != nil {
		return nil, err
	}
	return &model.ExportEnvelope{
		Version:    "1.0",
		Module:     "friend_rss",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Count:      len(feeds),
		Items: map[string]interface{}{
			"friend_rss":      feeds,
			"friend_rss_post": posts,
		},
	}, nil
}
