package friendsRepositories

import (
	"blog_api/src/model"
	"log"

	"gorm.io/gorm"
)

// InsertRssPost inserts a post unless another post already owns the same link.
func InsertRssPost(db *gorm.DB, post *model.RssPost) error {
	result := db.Where(model.RssPost{Link: post.Link}).FirstOrCreate(post)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return nil
	}

	log.Printf("已插入新文章: %s", post.Title)
	return nil
}

// GetPosts retrieves posts based on the provided query parameters.
func GetPosts(db *gorm.DB, query *model.PostQuery) ([]model.RssPost, int, error) {
	var posts []model.RssPost
	var total int64

	baseTx := db.Table("friend_rss_post AS p")
	if query.FriendLinkID != nil {
		baseTx = baseTx.Joins("JOIN friend_rss r ON p.rss_id = r.id").Where("r.friend_link_id = ?", *query.FriendLinkID)
	}
	if query.RssID != nil {
		baseTx = baseTx.Where("p.rss_id = ?", *query.RssID)
	}

	if err := baseTx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataTx := baseTx.Select("p.id, p.rss_id, p.title, p.link, p.description, p.author, p.time")
	if query.Page > 0 && query.PageSize > 0 {
		offset := (query.Page - 1) * query.PageSize
		dataTx = dataTx.Limit(query.PageSize).Offset(offset)
	}

	if err := dataTx.Order("p.time DESC").Order("p.id DESC").Scan(&posts).Error; err != nil {
		return nil, 0, err
	}
	for i := range posts {
		if posts[i].Time < 0 {
			posts[i].Time = 0
		}
	}

	return posts, int(total), nil
}
