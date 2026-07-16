package crawlerService

import (
	"blog_api/src/model"
	imageRepositories "blog_api/src/repositories/image"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const imageHealthBatchSize = 256

// CheckImagesHealth checks local and remote images and updates status when broken or recovered.
func CheckImagesHealth(db *gorm.DB) {
	client := &http.Client{Timeout: 10 * time.Second}
	checkedCount := 0
	brokenCount := 0
	recoveredCount := 0
	var mu sync.Mutex

	check := func(img model.Image) {
		exists, checked, err := checkImageExists(img, client)
		if err != nil {
			log.Printf("[crawler][image][WARN] Image check failed (ID: %d, URL: %s): %v", img.ID, img.URL, err)
			return
		}
		if !checked {
			return
		}

		mu.Lock()
		checkedCount++
		mu.Unlock()

		if !exists {
			if img.Status != "broken" {
				if err := imageRepositories.UpdateImage(db, &model.Image{ID: img.ID, Status: "broken"}); err != nil && err != gorm.ErrRecordNotFound {
					log.Printf("[crawler][image][WARN] Failed to mark broken image (ID: %d): %v", img.ID, err)
				} else {
					mu.Lock()
					brokenCount++
					mu.Unlock()
				}
			}
			return
		}

		if img.Status == "broken" {
			if err := imageRepositories.UpdateImage(db, &model.Image{ID: img.ID, Status: "normal"}); err != nil && err != gorm.ErrRecordNotFound {
				log.Printf("[crawler][image][WARN] Failed to mark recovered image (ID: %d): %v", img.ID, err)
			} else {
				mu.Lock()
				recoveredCount++
				mu.Unlock()
			}
		}
	}

	afterID := 0
	for {
		images, err := imageRepositories.ListImagesAfterID(db, afterID, imageHealthBatchSize)
		if err != nil {
			log.Printf("[crawler][image][ERR] Failed to list images: %v", err)
			return
		}
		if len(images) == 0 {
			break
		}
		CheckImagesConcurrently(images, check)
		afterID = images[len(images)-1].ID
		if len(images) < imageHealthBatchSize {
			break
		}
	}

	log.Printf("[crawler][image] Image health check finished. checked=%d broken=%d recovered=%d", checkedCount, brokenCount, recoveredCount)
}

func checkImageExists(img model.Image, client *http.Client) (bool, bool, error) {
	if img.IsLocal == 1 || img.LocalPath != "" {
		if img.LocalPath == "" {
			return false, false, fmt.Errorf("local image has empty local_path")
		}
		_, err := os.Stat(img.LocalPath)
		if err == nil {
			return true, true, nil
		}
		if os.IsNotExist(err) {
			return false, true, nil
		}
		return false, true, err
	}

	if img.IsOss == 1 || strings.HasPrefix(img.URL, "http://") || strings.HasPrefix(img.URL, "https://") {
		if img.URL == "" {
			return false, false, fmt.Errorf("remote image has empty url")
		}
		req, err := http.NewRequest(http.MethodHead, img.URL, nil)
		if err != nil {
			return false, true, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return false, true, err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			return true, true, nil
		}
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return false, true, nil
		}
		return false, true, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return false, false, fmt.Errorf("unknown image storage type")
}
