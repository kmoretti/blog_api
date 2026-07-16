package imageRepositories

import (
	"blog_api/src/model"
	"log"
	"math/rand/v2"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BatchInsertImages 批量插入图片信息到数据库
// 使用 OnConflict 来避免插入重复的 URL
func BatchInsertImages(db *gorm.DB, images []model.Image) error {
	if len(images) == 0 {
		log.Println("[db][image] No images to insert.")
		return nil
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoNothing: true,
	}).CreateInBatches(&images, 256)

	if result.Error != nil {
		log.Printf("[db][image][ERR] 无法批量插入图片: %v", result.Error)
		return result.Error
	}

	log.Printf("[db][image] 成功插入 %d 条图片记录", result.RowsAffected)
	return nil
}

// QueryImages 根据提供的选项查询图片，并返回分页结果和总数
func QueryImages(db *gorm.DB, opts model.ImageQueryOptions) (model.QueryImageResponse, error) {
	var resp model.QueryImageResponse
	query := db.Model(&model.Image{})

	// Apply status filter
	if opts.Status != "" {
		query = query.Where("status = ?", opts.Status)
	}

	if opts.Name != "" {
		query = query.Where("name LIKE ?", "%"+opts.Name+"%")
	}
	if err := query.Count(&resp.Total).Error; err != nil {
		return resp, err
	}
	if opts.Page > 0 && opts.PageSize > 0 {
		offset := (opts.Page - 1) * opts.PageSize
		query = query.Offset(offset).Limit(opts.PageSize)
	}

	if err := query.Order("id desc").Find(&resp.Images).Error; err != nil {
		return resp, err
	}

	return resp, nil
}

// CreateImage inserts a single image record into the database.
func CreateImage(db *gorm.DB, image *model.Image) error {
	err := db.Create(image).Error
	if err != nil {
		log.Printf("[db][image][ERR] 无法创建图片: %v", err)
		return err
	}
	log.Printf("[db][image] 成功创建图片记录，ID: %d", image.ID)
	return nil
}

// UpdateImage updates an existing image record in the database.
func UpdateImage(db *gorm.DB, image *model.Image) error {
	updates := map[string]interface{}{}
	if image.Name != "" {
		updates["name"] = image.Name
	}
	if image.URL != "" {
		updates["url"] = image.URL
	}
	if image.LocalPath != "" {
		updates["local_path"] = image.LocalPath
	}
	if image.IsLocal != 0 {
		updates["is_local"] = image.IsLocal
	}
	if image.IsOss != 0 {
		updates["is_oss"] = image.IsOss
	}
	if image.Status != "" {
		updates["status"] = image.Status
	}

	if len(updates) == 0 {
		log.Printf("[db][image][WARN] 没有可更新的字段，ID: %d", image.ID)
		return nil
	}

	result := db.Model(&model.Image{}).Where("id = ?", image.ID).Updates(updates)

	if result.RowsAffected == 0 {
		log.Printf("[db][image][WARN] 未找到要更新的图片，ID: %d", image.ID)
		return gorm.ErrRecordNotFound
	}

	log.Printf("[db][image] 成功更新图片记录，ID: %d", image.ID)
	return nil
}

// DeleteImage deletes an image record from the database by its ID.
func DeleteImage(db *gorm.DB, id int) error {
	result := db.Delete(&model.Image{}, id)
	if result.RowsAffected == 0 {
		log.Printf("[db][image][WARN] 未找到要删除的图片，ID: %d", id)
		return gorm.ErrRecordNotFound
	}

	log.Printf("[db][image] 成功删除图片记录，ID: %d", id)
	return nil
}

// GetImageByID retrieves a single image by its ID.
func GetImageByID(db *gorm.DB, id int) (*model.Image, error) {
	var image model.Image
	err := db.First(&image, id).Error
	if err != nil {
		log.Printf("[db][image][ERR] 无法通过ID %d 找到图片: %v", id, err)
		return nil, err
	}
	return &image, nil
}

// GetRandomImage retrieves a random image from the database.
func GetRandomImage(db *gorm.DB) (*model.Image, error) {
	var count int64
	query := db.Model(&model.Image{}).Where("status = ?", "normal")
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var image model.Image
	offset := rand.Int64N(count)
	err := query.Order("id ASC").Offset(int(offset)).First(&image).Error
	if err != nil {
		log.Printf("[db][image][ERR] 无法获取随机图片: %v", err)
		return nil, err
	}
	return &image, nil
}

// ListImagesAfterID retrieves a bounded image batch ordered by ID.
func ListImagesAfterID(db *gorm.DB, afterID, limit int) ([]model.Image, error) {
	var images []model.Image
	if limit <= 0 {
		return images, nil
	}
	if err := db.Where("id > ?", afterID).Order("id ASC").Limit(limit).Find(&images).Error; err != nil {
		log.Printf("[db][image][ERR] 无法获取图片列表: %v", err)
		return nil, err
	}
	return images, nil
}
