package friendsRepositories

import (
	"blog_api/src/model"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InsertFriendLinks inserts friend links from the configuration if they don't already exist.
func InsertFriendLinks(db *gorm.DB, friendLinks []model.FriendWebsite) error {
	var count int64
	if err := db.Model(&model.FriendWebsite{}).Count(&count).Error; err != nil {
		log.Printf("[db][friend][ERR]无法检查友链是否存在: %v", err)
		return err
	}

	if count > 0 {
		log.Println("[db][friend][init]检测到已有友链，跳过初始化")
		return nil
	}

	if len(friendLinks) == 0 {
		log.Println("No friend links to insert.")
		return nil
	}

	log.Println("[db][friend][init]Start inserting friend links...")

	for _, link := range friendLinks {
		newLink := model.FriendWebsite{
			Name:      link.Name,
			Link:      link.Link,
			Avatar:    link.Avatar,
			Info:      link.Info,
			Status:    "survival",
			EnableRss: true,
		}

		result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&newLink)
		if result.Error != nil {
			log.Printf("[db][friend][ERR]无法插入友链 %s: %v", link.Name, result.Error)
			continue
		}

		if result.RowsAffected == 0 {
			log.Printf("[db][friend][init]友链 %s 已存在，跳过", link.Name)
			continue
		}

		log.Printf("[db][friend][init]已插入友链: %s", link.Name)
	}

	log.Println("[db][friend][init]Friend links insertion process completed.")
	return nil
}

// QueryFriendLinks provides a unified interface for querying friend links.
func QueryFriendLinks(db *gorm.DB, opts model.FriendLinkQueryOptions) (model.QueryFriendLinksResponse, error) {
	var resp model.QueryFriendLinksResponse
	baseQuery := db.Model(&model.FriendWebsite{})

	// Apply status filters
	if opts.Status != "" {
		baseQuery = baseQuery.Where("status = ?", opts.Status)
	}
	if len(opts.Statuses) > 0 {
		if opts.NotIn {
			baseQuery = baseQuery.Where("status NOT IN ?", opts.Statuses)
		} else {
			baseQuery = baseQuery.Where("status IN ?", opts.Statuses)
		}
	}

	if opts.IsDied != nil {
		baseQuery = baseQuery.Where("is_died = ?", *opts.IsDied)
	}

	// Apply search filter
	if opts.Search != "" {
		searchPattern := "%" + opts.Search + "%"
		baseQuery = baseQuery.Where("website_name LIKE ? OR website_url LIKE ? OR description LIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if opts.Count {
		if err := baseQuery.Count(&resp.Count).Error; err != nil {
			return resp, fmt.Errorf("could not count friend links: %w", err)
		}
		return resp, nil
	}

	if err := baseQuery.Count(&resp.Count).Error; err != nil {
		return resp, fmt.Errorf("could not count friend links for pagination: %w", err)
	}

	query := baseQuery.Order("updated_at DESC")

	// Apply pagination and ordering
	if opts.Limit > 0 {
		query = query.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		query = query.Offset(opts.Offset)
	}
	selectFields := "id, website_name, website_url, website_icon_url, description, email, times, status, is_died, enable_rss, updated_at"
	if err := query.Select(selectFields).Find(&resp.Links).Error; err != nil {
		return resp, fmt.Errorf("could not query friend links: %w", err)
	}

	return resp, nil
}

// GetFriendLinkByID fetches a single friend link by ID.
func GetFriendLinkByID(db *gorm.DB, id int) (model.FriendWebsite, error) {
	var link model.FriendWebsite
	selectFields := "id, website_name, website_url, website_icon_url, description, email, times, status, is_died, enable_rss, updated_at"
	err := db.Model(&model.FriendWebsite{}).Select(selectFields).Where("id = ?", id).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.FriendWebsite{}, err
		}
		return model.FriendWebsite{}, fmt.Errorf("could not query friend link by id %d: %w", id, err)
	}
	return link, nil
}

// GetFriendLinkByEmail fetches a single friend link by email.
func GetFriendLinkByEmail(db *gorm.DB, email string) (model.FriendWebsite, error) {
	var link model.FriendWebsite
	selectFields := "id, website_name, website_url, website_icon_url, description, email, times, status, is_died, enable_rss, updated_at"
	err := db.Model(&model.FriendWebsite{}).Select(selectFields).Where("email = ?", email).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.FriendWebsite{}, err
		}
		return model.FriendWebsite{}, fmt.Errorf("could not query friend link by email %s: %w", email, err)
	}
	return link, nil
}

// UpdateFriendLink updates the details of a friend link after crawling.
func UpdateFriendLink(db *gorm.DB, link model.FriendWebsite, result model.CrawlResult) error {
	success := result.Status == "survival"
	var reachedThreshold bool
	link.Times, link.Status, reachedThreshold = model.ComputeFailureState(
		link.Times,
		success,
		4,
		"survival",
		result.Status,
		"",
	)
	if success {
		link.IsDied = false
	} else if reachedThreshold {
		link.IsDied = true
	}

	if result.RedirectURL != "" {
		link.Link = result.RedirectURL
	}

	updates := map[string]interface{}{
		"website_url": link.Link,
		"description": gorm.Expr("CASE WHEN description = '' THEN ? ELSE description END", result.Description),
		"status":      link.Status,
		"times":       link.Times,
		"is_died":     link.IsDied,
	}

	// 仅当现有 icon 为空时才覆盖，避免已有 icon 被新结果替换
	if link.Avatar == "" && result.IconURL != "" {
		updates["website_icon_url"] = resolveAvatarURL(result.IconURL, link.Link)
	}

	if err := db.Model(&model.FriendWebsite{}).Where("id = ?", link.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("could not update friend link with id %d: %w", link.ID, err)
	}

	log.Printf("为 ID  %d 更新友链. 状态: %s, 时间: %d, is_died: %t", link.ID, link.Status, link.Times, link.IsDied)
	return nil
}

// CreateFriendLink inserts a single new friend link into the database.
func CreateFriendLink(db *gorm.DB, link model.FriendWebsite) (int64, error) {
	newLink := model.FriendWebsite{
		Name:      link.Name,
		Link:      link.Link,
		Avatar:    link.Avatar,
		Info:      link.Info,
		Email:     link.Email,
		Status:    "pending",
		EnableRss: link.EnableRss,
	}

	if err := db.Create(&newLink).Error; err != nil {
		return 0, fmt.Errorf("could not execute insert statement for friend link: %w", err)
	}

	log.Printf("[db][friend] 已插入新友链: %s，ID 为: %d", link.Name, newLink.ID)
	return int64(newLink.ID), nil
}

// DeleteFriendLinkByID deletes a friend link by its ID and returns the deleted link.
func DeleteFriendLinkByID(db *gorm.DB, id uint) (model.FriendWebsite, error) {
	var deletedLink model.FriendWebsite
	var rowsDeleted int64
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).First(&deletedLink).Error; err != nil {
			return fmt.Errorf("could not query friend link for deletion: %w", err)
		}

		res := tx.Delete(&deletedLink)
		if res.Error != nil {
			return fmt.Errorf("could not delete friend link: %w", res.Error)
		}
		rowsDeleted = res.RowsAffected
		return nil
	})
	if err != nil {
		return model.FriendWebsite{}, err
	}

	log.Printf("[db][friend] 已删除 %d 个友链", rowsDeleted)

	return deletedLink, nil
}

// UpdateFriendLinkByID updates a friend link by its ID and handles cascading deletes for RSS data if necessary.
func UpdateFriendLinkByID(db *gorm.DB, id uint, req model.EditFriendLinkReq) (int64, error) {
	if len(req.Data) == 0 {
		return 0, fmt.Errorf("no data provided for update")
	}

	// Whitelist of updatable columns
	updatableColumns := map[string]bool{
		"website_name":     true,
		"website_url":      true,
		"website_icon_url": true,
		"description":      true,
		"email":            true,
		"times":            true,
		"status":           true,
		"enable_rss":       true,
		"is_died":          true,
	}

	updates := map[string]interface{}{}
	for col, val := range req.Data {
		if !updatableColumns[col] {
			continue
		}
		if !req.Opt.OverwriteIfBlank {
			if s, ok := val.(string); ok && s == "" {
				continue
			}
		}
		updates[col] = val
	}

	if len(updates) == 0 {
		log.Println("[db][friend] No valid fields to update after filtering.")
		return 0, nil
	}

	// Check if enable_rss is being set to false
	disableRss := false
	if val, ok := updates["enable_rss"].(bool); ok && !val {
		disableRss = true
	}

	var rowsAffected int64
	err := db.Transaction(func(tx *gorm.DB) error {
		// If disabling RSS, delete related data first
		if disableRss {
			if err := DeleteRssDataByFriendLinkID(tx, int(id)); err != nil {
				return err
			}
		}

		// Perform the update
		result := tx.Model(&model.FriendWebsite{}).Where("id = ?", id).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("could not execute update for friend link id %d: %w", id, result.Error)
		}
		rowsAffected = result.RowsAffected
		return nil
	})

	if err != nil {
		return 0, err
	}

	log.Printf("[db][friend] 为 ID: %d 更新友链. Rows affected: %d", id, rowsAffected)
	return rowsAffected, nil
}

// FriendLinkExists checks if a friend link with the given ID exists.
func FriendLinkExists(db *gorm.DB, id int) (bool, error) {
	var count int64
	if err := db.Model(&model.FriendWebsite{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return false, fmt.Errorf("could not check for existing friend_link: %w", err)
	}
	return count > 0, nil
}
