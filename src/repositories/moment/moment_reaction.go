package momentRepositories

import (
	"blog_api/src/model"

	"gorm.io/gorm"
)

// CreateMomentReaction inserts a new reaction.
func CreateMomentReaction(db *gorm.DB, reaction *model.MomentReaction) error {
	return db.Create(reaction).Error
}

// DeleteMomentReaction removes a reaction by moment, fingerprint, and reaction type.
func DeleteMomentReaction(db *gorm.DB, momentID, fingerprintID int, reaction string) error {
	result := db.Where(
		"moment_id = ? AND fingerprint_id = ? AND reaction = ?",
		momentID,
		fingerprintID,
		reaction,
	).Delete(&model.MomentReaction{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

type momentReactionCount struct {
	MomentID int    `gorm:"column:moment_id"`
	Reaction string `gorm:"column:reaction"`
	Count    int    `gorm:"column:count"`
}

// GetReactionCountsForMoments returns reaction counts grouped by moment and reaction.
func GetReactionCountsForMoments(db *gorm.DB, momentIDs []int) (map[int]map[string]int, error) {
	result := make(map[int]map[string]int)
	if len(momentIDs) == 0 {
		return result, nil
	}

	var rows []momentReactionCount
	if err := db.Model(&model.MomentReaction{}).
		Select("moment_id, reaction, COUNT(*) as count").
		Where("moment_id IN ?", momentIDs).
		Group("moment_id, reaction").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if _, ok := result[row.MomentID]; !ok {
			result[row.MomentID] = make(map[string]int)
		}
		result[row.MomentID][row.Reaction] = row.Count
	}

	return result, nil
}

type momentUserReaction struct {
	MomentID int    `gorm:"column:moment_id"`
	Reaction string `gorm:"column:reaction"`
}

// GetUserReactionsForMoments returns the user's selected reaction per moment.
func GetUserReactionsForMoments(db *gorm.DB, momentIDs []int, fingerprintID int) (map[int]string, error) {
	result := make(map[int]string)
	if len(momentIDs) == 0 || fingerprintID <= 0 {
		return result, nil
	}

	var rows []momentUserReaction
	if err := db.Model(&model.MomentReaction{}).
		Select("moment_id, reaction").
		Where("moment_id IN ? AND fingerprint_id = ?", momentIDs, fingerprintID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if _, ok := result[row.MomentID]; !ok {
			result[row.MomentID] = row.Reaction
		}
	}

	return result, nil
}

// ClearReactionsByType removes all reactions of a specific type from a moment.
func ClearReactionsByType(db *gorm.DB, momentID int, reaction string) error {
	return db.Where(
		"moment_id = ? AND reaction = ?",
		momentID,
		reaction,
	).Delete(&model.MomentReaction{}).Error
}
