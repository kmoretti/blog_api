package service

import (
	"blog_api/src/config"
	"blog_api/src/model"
	imageRepositories "blog_api/src/repositories/image"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"gorm.io/gorm"
)

// ScanAndSaveImages 扫描指定目录下的图片，根据配置进行格式转换，并将其信息保存到数据库
func ScanAndSaveImages(db *gorm.DB) error {
	cfg := config.GetConfig()
	imagePath := cfg.Data.Image.Path
	if imagePath == "" {
		log.Println("[service][image] Image path is not configured, skipping scan.")
		return nil
	}

	var images []model.Image
	targetFormat := strings.ToLower(cfg.Data.Image.ConvTo)
	walkErr := filepath.Walk(imagePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("[service][image][WARN] Error accessing path %s: %v. Skipping.", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		// 处理单个图片文件
		processed, err := processSingleImage(path, imagePath, targetFormat)
		if err != nil {
			log.Printf("[service][image][DEBUG] Skipping file %s: %v", path, err)
			return nil
		}

		if processed != nil {
			if processed.converted {
				updated, err := updateImageAfterConvert(db, processed.originalURL, processed.originalPath, processed.image)
				if err != nil {
					log.Printf("[service][image][WARN] Failed to update image record after conversion: %v", err)
				} else if updated {
					return nil
				}
			}
			images = append(images, *processed.image)
		}
		return nil
	})

	if walkErr != nil {
		log.Printf("[service][image][ERR] Failed to walk through image directory: %v", walkErr)
		return walkErr
	}

	if len(images) > 0 {
		log.Printf("[service][image] Found and processed %d images.", len(images))
		newImages, err := imageRepositories.FilterNonExistingImages(db, images)
		if err != nil {
			log.Printf("[service][image][ERR] Failed to filter existing images: %v", err)
			return err
		}

		if len(newImages) == 0 {
			log.Println("[service][image] All images already exist in database. Skipping insert.")
			return nil
		}

		log.Printf("[service][image] Inserting %d new images...", len(newImages))
		return imageRepositories.BatchInsertImages(db, newImages)
	}

	log.Println("[service][image] No new images found to process.")
	return nil
}

type processedImage struct {
	image        *model.Image
	originalPath string
	originalURL  string
	converted    bool
}

// processSingleImage 处理单个图片文件：检查格式、转换（如果需要）、构建模型对象
func processSingleImage(filePath, rootPath, targetFormat string) (*processedImage, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if !isSupportedImage(ext) {
		return nil, fmt.Errorf("unsupported image format")
	}

	originalRelPath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		log.Printf("[service][image][WARN] Failed to get relative path for %s: %v", filePath, err)
		return nil, err
	}
	originalURL := filepath.ToSlash(filepath.Join("/image", originalRelPath))
	finalPath := filePath

	if targetFormat != "" && ext != "."+targetFormat {
		newPath := strings.TrimSuffix(filePath, ext) + "." + targetFormat

		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			if err := convertImage(filePath, newPath, targetFormat); err != nil {
				log.Printf("[service][image][WARN] Failed to convert %s: %v", filePath, err)
				return nil, err
			}
			log.Printf("[service][image] Converted %s to %s", filePath, newPath)

			if err := os.Remove(filePath); err != nil {
				log.Printf("[service][image][WARN] Failed to remove original file %s: %v", filePath, err)
			} else {
				log.Printf("[service][image] Removed original file: %s", filePath)
			}
		}
		finalPath = newPath
	}
	converted := finalPath != filePath

	// 构建相对路径和 URL
	relPath, err := filepath.Rel(rootPath, finalPath)
	if err != nil {
		log.Printf("[service][image][WARN] Failed to get relative path for %s: %v", finalPath, err)
		return nil, err
	}
	url := filepath.ToSlash(filepath.Join("/image", relPath))

	return &processedImage{
		image: &model.Image{
			Name:      filepath.Base(finalPath),
			URL:       url,
			LocalPath: finalPath,
			IsLocal:   1,
			Status:    "normal",
		},
		originalPath: filePath,
		originalURL:  originalURL,
		converted:    converted,
	}, nil
}

// isSupportedImage 检查文件扩展名是否为支持的图片格式
func isSupportedImage(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// convertImage 将源图片转换为目标格式并保存
func convertImage(srcPath, destPath, targetFormat string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("could not open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("could not create destination file: %w", err)
	}
	defer destFile.Close()

	if targetFormat == "webp" {
		_, err := io.Copy(destFile, srcFile)
		return err
	}

	img, _, err := image.Decode(srcFile)
	if err != nil {
		return fmt.Errorf("could not decode image: %w", err)
	}

	switch targetFormat {
	case "jpg", "jpeg":
		return imaging.Encode(destFile, img, imaging.JPEG, imaging.JPEGQuality(85))
	case "png":
		return imaging.Encode(destFile, img, imaging.PNG)
	case "gif":
		return imaging.Encode(destFile, img, imaging.GIF)
	default:
		return fmt.Errorf("unsupported target format: %s", targetFormat)
	}
}

func updateImageAfterConvert(db *gorm.DB, originalURL, originalPath string, img *model.Image) (bool, error) {
	if img == nil || (originalURL == "" && originalPath == "") {
		return false, nil
	}
	updates := map[string]interface{}{
		"name":       img.Name,
		"url":        img.URL,
		"local_path": img.LocalPath,
		"is_local":   img.IsLocal,
		"is_oss":     img.IsOss,
	}
	result := db.Model(&model.Image{}).
		Where("url = ? OR local_path = ?", originalURL, originalPath).
		Updates(updates)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("[service][image] Updated image record after conversion: %s -> %s", originalURL, img.URL)
		return true, nil
	}
	return false, nil
}
