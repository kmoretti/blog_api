package repositories

import (
	"blog_api/src/model"

	"gorm.io/gorm"
)

// GetSystemStats retrieves all system statistics.
func GetSystemStats(db *gorm.DB) (model.StatusData, error) {
	var stats model.StatusData
	var pieData []model.FriendLinkStatusCount
	var monthlyData []model.RssPostCountMonthly

	// Get basic counts
	countsQuery := `
		SELECT
			(SELECT COUNT(*) FROM friend_link) AS friend_link_count,
			(SELECT COUNT(*) FROM friend_rss) AS rss_count,
			(SELECT COUNT(*) FROM friend_rss_post) AS rss_post_count
	`
	var counts struct {
		FriendLinkCount int
		RssCount        int
		RssPostCount    int
	}

	if err := db.Raw(countsQuery).Scan(&counts).Error; err != nil {
		return stats, err
	}

	stats.FriendLinkCount = counts.FriendLinkCount
	stats.RssCount = counts.RssCount
	stats.RssPostCount = counts.RssPostCount

	// Get friend link status distribution
	pieQuery := `
		SELECT status, COUNT(*) as count
		FROM friend_link
		GROUP BY status
	`
	if err := db.Raw(pieQuery).Scan(&pieData).Error; err != nil {
		return stats, err
	}
	stats.FriendLinkStatusPie = pieData

	// Get monthly RSS post count (for the last 12 months)
	monthlyQuery := `
		SELECT strftime('%Y-%m', datetime(time, 'unixepoch')) as month, COUNT(*) as count
		FROM friend_rss_post
		WHERE time >= strftime('%s', date('now', '-12 months'))
		GROUP BY month
		ORDER BY month
	`
	if err := db.Raw(monthlyQuery).Scan(&monthlyData).Error; err != nil {
		return stats, err
	}
	stats.RssPostCountMonthly = monthlyData

	return stats, nil
}
