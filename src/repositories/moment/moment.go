package momentRepositories

import (
	"blog_api/src/model"

	"gorm.io/gorm"
)

// QueryMoments retrieves moments based on pagination and returns the list and total count.
func QueryMoments(db *gorm.DB, page, pageSize int, status string) ([]model.Moment, int64, error) {
	var moments []model.Moment
	var total int64

	query := db.Model(&model.Moment{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Order("created_at DESC").Order("id DESC").Find(&moments).Error; err != nil {
		return nil, 0, err
	}

	return moments, total, nil
}

// GetMediaForMoments retrieves media files for a list of moment IDs.
func GetMediaForMoments(db *gorm.DB, momentIDs []int) ([]model.MomentMedia, error) {
	var media []model.MomentMedia
	if len(momentIDs) == 0 {
		return media, nil
	}
	if err := db.Where("moment_id IN ? AND is_deleted = 0", momentIDs).
		Order("moment_id ASC").
		Order("id ASC").
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

// CreateMoment creates a new moment and its associated media in a transaction.
func CreateMoment(db *gorm.DB, moment *model.Moment, media []model.MomentMedia) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(moment).Error; err != nil {
			return err
		}

		if len(media) > 0 {
			for i := range media {
				media[i].MomentID = moment.ID
			}
			if err := tx.Create(&media).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// DeleteMoment deletes a moment.
func DeleteMoment(db *gorm.DB, id int) error {
	return db.Delete(&model.Moment{}, id).Error
}

// UpdateMoment updates fields for a moment.
func UpdateMoment(db *gorm.DB, id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	result := db.Model(&model.Moment{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetMomentByID retrieves a moment by ID.
func GetMomentByID(db *gorm.DB, id int) (*model.Moment, error) {
	var moment model.Moment
	if err := db.First(&moment, id).Error; err != nil {
		return nil, err
	}
	return &moment, nil
}

// DeleteMomentByChannelMessage deletes a moment using channel_id and message_id.
func DeleteMomentByChannelMessage(db *gorm.DB, channelID, messageID int64) error {
	result := db.Where("channel_id = ? AND message_id = ?", channelID, messageID).Delete(&model.Moment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MomentExistsByChannelMessage checks if a moment already exists for a channel/message pair.
func MomentExistsByChannelMessage(db *gorm.DB, channelID, messageID int64) (bool, error) {
	var id int
	err := db.Model(&model.Moment{}).
		Select("id").
		Where("channel_id = ? AND message_id = ?", channelID, messageID).
		Limit(1).
		Scan(&id).Error
	if err != nil {
		return false, err
	}
	return id > 0, nil
}
