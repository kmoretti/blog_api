package friendsRepositories

import (
	"blog_api/src/model"
	"blog_api/src/service/friendlink"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	defaultGroupName        = "网上邻居"
	defaultGroupDescription = "我的网上邻居~"
)

var (
	ErrFriendLinkGroupNotFound  = errors.New("friend link group not found")
	ErrInvalidFriendLinkGroupID = errors.New("invalid friend link group id")
)

// EnsureDefaultFriendLinkGroup creates the default group if it does not exist.
// It returns the default group ID.
func EnsureDefaultFriendLinkGroup(db *gorm.DB) (int, error) {
	var group model.FriendLinkGroup
	err := db.Where("name = ?", defaultGroupName).First(&group).Error
	if err == nil {
		if group.Description == "" {
			if err := db.Model(&group).Update("description", defaultGroupDescription).Error; err != nil {
				return 0, err
			}
		}
		return group.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	now := time.Now().Unix()
	group = model.FriendLinkGroup{
		Name:        defaultGroupName,
		Description: defaultGroupDescription,
		SortOrder:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(&group).Error; err != nil {
		return 0, err
	}
	return group.ID, nil
}

// CreateFriendLinkGroup creates a new friend link group.
func CreateFriendLinkGroup(db *gorm.DB, group *model.FriendLinkGroup) error {
	if group.Name == "" {
		return fmt.Errorf("group name is required")
	}
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	return db.Create(group).Error
}

// UpdateFriendLinkGroup updates an existing friend link group.
func UpdateFriendLinkGroup(db *gorm.DB, id int, group *model.FriendLinkGroup) error {
	if id <= 0 {
		return fmt.Errorf("invalid group id")
	}
	if group.Name == "" {
		return fmt.Errorf("group name is required")
	}
	updates := map[string]interface{}{
		"name":        group.Name,
		"description": group.Description,
		"sort_order":  group.SortOrder,
		"updated_at":  time.Now().Unix(),
	}
	result := db.Model(&model.FriendLinkGroup{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFriendLinkGroupNotFound
	}
	return nil
}

// DeleteFriendLinkGroup deletes a friend link group and removes its mappings.
// Members are not deleted; they become ungrouped and will fall into the default group.
func DeleteFriendLinkGroup(db *gorm.DB, id int) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var group model.FriendLinkGroup
		if err := tx.Where("id = ?", id).First(&group).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrFriendLinkGroupNotFound
			}
			return err
		}
		if err := tx.Where("friend_link_group_id = ?", id).Delete(&model.FriendLinkGroupMapping{}).Error; err != nil {
			return err
		}
		return tx.Delete(&group).Error
	})
}

// GetFriendLinkGroupByID fetches a single group by ID.
func GetFriendLinkGroupByID(db *gorm.DB, id int) (model.FriendLinkGroup, error) {
	var group model.FriendLinkGroup
	err := db.Where("id = ?", id).First(&group).Error
	return group, err
}

// ListFriendLinkGroups returns all groups ordered by sort_order and id.
func ListFriendLinkGroups(db *gorm.DB) ([]model.FriendLinkGroup, error) {
	var groups []model.FriendLinkGroup
	err := db.Order("sort_order ASC, id ASC").Find(&groups).Error
	return groups, err
}

// SetFriendLinkGroups replaces the groups a friend link belongs to.
func SetFriendLinkGroups(db *gorm.DB, friendLinkID int, groupIDs []int) error {
	if friendLinkID <= 0 {
		return fmt.Errorf("invalid friend link id")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		mappings := make([]model.FriendLinkGroupMapping, 0, len(groupIDs))
		seen := make(map[int]struct{}, len(groupIDs))
		for _, gid := range groupIDs {
			if gid <= 0 {
				return ErrInvalidFriendLinkGroupID
			}
			var count int64
			if err := tx.Model(&model.FriendLinkGroup{}).Where("id = ?", gid).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrFriendLinkGroupNotFound
			}
			if _, ok := seen[gid]; ok {
				continue
			}
			seen[gid] = struct{}{}
			mappings = append(mappings, model.FriendLinkGroupMapping{
				FriendLinkID:      friendLinkID,
				FriendLinkGroupID: gid,
			})
		}
		if err := tx.Where("friend_link_id = ?", friendLinkID).Delete(&model.FriendLinkGroupMapping{}).Error; err != nil {
			return err
		}
		if len(mappings) == 0 {
			return nil
		}
		return tx.Create(&mappings).Error
	})
}

// GetFriendLinkGroupIDs returns the group IDs associated with a friend link.
func GetFriendLinkGroupIDs(db *gorm.DB, friendLinkID int) ([]int, error) {
	var mappings []model.FriendLinkGroupMapping
	if err := db.Where("friend_link_id = ?", friendLinkID).Find(&mappings).Error; err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(mappings))
	for _, m := range mappings {
		ids = append(ids, m.FriendLinkGroupID)
	}
	return ids, nil
}

// MigrateExistingFriendLinksToDefaultGroup moves all friend links without a group
// mapping into the default group and assigns a random color to survival links
// that do not have one yet. Safe to call multiple times.
func MigrateExistingFriendLinksToDefaultGroup(db *gorm.DB) error {
	defaultID, err := EnsureDefaultFriendLinkGroup(db)
	if err != nil {
		return err
	}

	var linkIDs []int
	if err := db.Model(&model.FriendWebsite{}).
		Select("id").
		Where("id NOT IN (?)", db.Model(&model.FriendLinkGroupMapping{}).Select("friend_link_id")).
		Find(&linkIDs).Error; err != nil {
		return err
	}

	if len(linkIDs) > 0 {
		mappings := make([]model.FriendLinkGroupMapping, 0, len(linkIDs))
		for _, id := range linkIDs {
			mappings = append(mappings, model.FriendLinkGroupMapping{
				FriendLinkID:      id,
				FriendLinkGroupID: defaultID,
			})
		}
		if err := db.Create(&mappings).Error; err != nil {
			return err
		}
	}

	links, err := ListFriendLinksWithoutColor(db)
	if err != nil {
		return err
	}
	for _, link := range links {
		if err := UpdateFriendLinkColor(db, link.ID, friendlink.RandomColor()); err != nil {
			return err
		}
	}

	return nil
}
