package friendsRepositories

import (
	"blog_api/src/model"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// QueryFriendRss provides a unified interface for querying friend RSS feeds.
func QueryFriendRss(db *gorm.DB, opts model.FriendRssQueryOptions) (model.QueryFriendRssResponse, error) {
	var resp model.QueryFriendRssResponse
	query := db.Model(&model.FriendRss{})

	if opts.Status != "" && opts.Status == "valid" {
		query = db.Table("friend_rss").
			Joins("JOIN friend_link ON friend_link.id = friend_rss.friend_link_id").
			Where("friend_link.is_died = ?", false).
			Where("friend_rss.is_died = ?", false).
			Where("friend_rss.status != ?", "pause")
		if opts.FriendLinkID > 0 {
			query = query.Where("friend_rss.friend_link_id = ?", opts.FriendLinkID)
		}
	} else {
		if opts.FriendLinkID > 0 {
			query = query.Where("friend_link_id = ?", opts.FriendLinkID)
		}
		if opts.Status != "" {
			query = query.Where("status = ?", opts.Status)
		}
	}
	if opts.IsDied != nil {
		if opts.Status == "valid" {
			query = query.Where("friend_rss.is_died = ?", *opts.IsDied)
		} else {
			query = query.Where("is_died = ?", *opts.IsDied)
		}
	}

	// Get total count
	if err := query.Count(&resp.Total).Error; err != nil {
		return resp, err
	}

	// Apply pagination
	if opts.Page > 0 && opts.PageSize > 0 {
		offset := (opts.Page - 1) * opts.PageSize
		query = query.Offset(offset).Limit(opts.PageSize)
	}

	query = query.Order("friend_rss.updated_at DESC").Order("friend_rss.id DESC")
	if err := query.Find(&resp.Feeds).Error; err != nil {
		return resp, err
	}

	return resp, nil
}

// CreateFriendRssFeeds creates a new friend_rss record, avoiding duplicates.
func CreateFriendRssFeeds(db *gorm.DB, friendLinkID int, rssURL string, name string) (*model.FriendRss, error) {
	if rssURL == "" {
		return nil, fmt.Errorf("rssURL cannot be empty")
	}

	var existing model.FriendRss
	err := db.Where("friend_link_id = ? AND rss_url = ?", friendLinkID, rssURL).First(&existing).Error
	if err == nil {
		log.Printf("RSS feed '%s' already exists for friend link ID %d, returning existing record.", rssURL, friendLinkID)
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newRSS := model.FriendRss{
		FriendLinkID: friendLinkID,
		RssURL:       rssURL,
		Name:         name,
		Times:        0,
		Status:       "survival",
		IsDied:       false,
	}
	if err := db.Create(&newRSS).Error; err != nil {
		return nil, err
	}

	log.Printf("Successfully inserted RSS feed '%s' for friend link ID %d.", rssURL, friendLinkID)
	return &newRSS, nil
}

// DeleteFriendRssByID deletes a friend_rss entry and all associated posts by its ID.
func DeleteFriendRssByID(db *gorm.DB, id uint) (int64, error) {
	var rowsAffected int64

	err := db.Transaction(func(tx *gorm.DB) error {
		// Delete associated posts first
		if err := tx.Where("rss_id = ?", id).Delete(&model.RssPost{}).Error; err != nil {
			return err
		}

		// GORM can delete with a primary key
		result := tx.Delete(&model.FriendRss{}, id)
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		return nil
	})

	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("成功删除 %d 个 RSS 源", rowsAffected)
	}

	return rowsAffected, nil
}

// DeleteRssDataByFriendLinkID deletes all RSS feeds and their posts for a given friend_link_id within a transaction.
func DeleteRssDataByFriendLinkID(tx *gorm.DB, friendLinkID int) error {
	// Find all RSS feeds associated with the friend link
	var rssFeeds []model.FriendRss
	if err := tx.Where("friend_link_id = ?", friendLinkID).Find(&rssFeeds).Error; err != nil {
		return err
	}

	if len(rssFeeds) == 0 {
		log.Printf("No RSS feeds to delete for friend_link_id %d", friendLinkID)
		return nil // Nothing to do
	}

	// Collect all RSS feed IDs
	rssIDs := make([]int, len(rssFeeds))
	for i, feed := range rssFeeds {
		rssIDs[i] = feed.ID
	}

	// Delete associated posts first (manual cascade for SQLite safety)
	if err := tx.Where("rss_id IN ?", rssIDs).Delete(&model.RssPost{}).Error; err != nil {
		return err
	}

	// Delete the RSS feeds themselves
	if err := tx.Where("friend_link_id = ?", friendLinkID).Delete(&model.FriendRss{}).Error; err != nil {
		return err
	}

	log.Printf("Successfully deleted %d RSS feeds and their posts for friend_link_id %d", len(rssFeeds), friendLinkID)
	return nil
}

// UpdateFriendRssByID updates a friend_rss entry by its ID.
func UpdateFriendRssByID(db *gorm.DB, id uint, req model.EditFriendRssReq) (int64, error) {
	if len(req.Data) == 0 {
		return 0, fmt.Errorf("no data provided for update")
	}

	// Whitelist of updatable columns
	updatableColumns := map[string]bool{
		"name":           true,
		"rss_url":        true,
		"status":         true,
		"friend_link_id": true,
		"is_died":        true,
	}

	updates := make(map[string]interface{})
	for col, val := range req.Data {
		if updatableColumns[col] {
			updates[col] = val
		}
	}
	if len(updates) == 0 {
		log.Println("[db][friend_rss] No valid fields to update after filtering.")
		return 0, nil
	}

	result := db.Model(&model.FriendRss{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return 0, result.Error
	}

	rowsAffected := result.RowsAffected

	log.Printf("[db][friend_rss] Updated friend_rss with ID: %d. Rows affected: %d", id, rowsAffected)
	return rowsAffected, nil
}
