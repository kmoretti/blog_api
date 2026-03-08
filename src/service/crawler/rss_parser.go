package crawlerService

import (
	"blog_api/src/config"
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"log"
	"net/http"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

const maxRssParseFailures = 4

func newRssParser() *gofeed.Parser {
	fp := gofeed.NewParser()
	timeoutSeconds := config.GetConfig().Crawler.RssTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	fp.Client = &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}
	return fp
}

// ParseRssFeed parses an RSS feed and saves the articles to the database.
func ParseRssFeed(db *gorm.DB, friendRssID int, rssURL string) {
	fp := newRssParser()
	feed, err := fp.ParseURL(rssURL)
	if err != nil {
		log.Printf("解析 RSS feed %s 时出错: %v", rssURL, err)
		updateRssParseState(db, friendRssID, false)
		return
	}
	updateRssParseState(db, friendRssID, true)

	friendRssName := ""
	var friendRss model.FriendRss
	if err := db.Select("name").Where("id = ?", friendRssID).First(&friendRss).Error; err == nil {
		friendRssName = friendRss.Name
	} else {
		log.Printf("获取 RSS 源名称失败 (id=%d): %v", friendRssID, err)
	}

	p := bluemonday.StripTagsPolicy()
	for _, item := range feed.Items {
		publishedTime := item.PublishedParsed
		if publishedTime == nil {
			// If PublishedParsed is nil, use UpdatedParsed
			publishedTime = item.UpdatedParsed
			if publishedTime == nil {
				log.Printf("跳过没有发布或更新时间的文章: %s", item.Title)
				continue
			}
		}

		time := publishedTime.Unix()
		if time < 0 {
			time = 0
		}

		author := ""
		if item.Author != nil {
			if item.Author.Name != "" {
				author = item.Author.Name
			} else if item.Author.Email != "" {
				author = item.Author.Email
			}
		}
		if author == "" && len(item.Authors) > 0 {
			for _, candidate := range item.Authors {
				if candidate == nil {
					continue
				}
				if candidate.Name != "" {
					author = candidate.Name
					break
				}
				if candidate.Email != "" {
					author = candidate.Email
					break
				}
			}
		}
		if author == "" {
			author = friendRssName
		}

		post := &model.RssPost{
			RssID:       friendRssID,
			Title:       item.Title,
			Link:        item.Link,
			Description: p.Sanitize(item.Description),
			Author:      author,
			Time:        time,
		}

		err := friendsRepositories.InsertRssPost(db, post)
		if err != nil {
			log.Printf("插入文章 '%s' 时出错: %v", item.Title, err)
		}
	}
}

func updateRssParseState(db *gorm.DB, friendRssID int, success bool) {
	var rss model.FriendRss
	if err := db.Select("id, times, status, is_died").Where("id = ?", friendRssID).First(&rss).Error; err != nil {
		log.Printf("更新 RSS 解析状态前查询失败 (id=%d): %v", friendRssID, err)
		return
	}

	newTimes, newStatus, reachedThreshold := model.ComputeFailureState(
		rss.Times,
		success,
		maxRssParseFailures,
		"survival",
		"timeout",
		"error",
	)
	newIsDied := rss.IsDied
	if !success && reachedThreshold {
		newIsDied = true
	}

	if rss.Times == newTimes && rss.Status == newStatus && rss.IsDied == newIsDied {
		return
	}

	if err := db.Model(&model.FriendRss{}).
		Where("id = ?", friendRssID).
		Updates(map[string]interface{}{
			"times":   newTimes,
			"status":  newStatus,
			"is_died": newIsDied,
		}).Error; err != nil {
		log.Printf("更新 RSS 解析状态失败 (id=%d): %v", friendRssID, err)
		return
	}

	log.Printf("RSS 解析状态更新 (id=%d, success=%t, times=%d, status=%s, is_died=%t)", friendRssID, success, newTimes, newStatus, newIsDied)
}

// GetRssTitle fetches and returns the title of an RSS feed.
func GetRssTitle(rssURL string) (string, error) {
	fp := newRssParser()
	feed, err := fp.ParseURL(rssURL)
	if err != nil {
		log.Printf("解析 RSS feed %s 时出错: %v", rssURL, err)
		return "", err
	}
	return feed.Title, nil
}

// CheckAndReviveRssFeed probes a died RSS feed and revives it on success.
func CheckAndReviveRssFeed(db *gorm.DB, friendRssID int, rssURL string) {
	fp := newRssParser()
	if _, err := fp.ParseURL(rssURL); err != nil {
		log.Printf("失效 RSS 探活失败 %s (id=%d): %v", rssURL, friendRssID, err)
		return
	}

	if err := db.Model(&model.FriendRss{}).
		Where("id = ?", friendRssID).
		Updates(map[string]interface{}{
			"times":   0,
			"status":  "survival",
			"is_died": false,
		}).Error; err != nil {
		log.Printf("失效 RSS 复活状态写入失败 (id=%d): %v", friendRssID, err)
		return
	}

	log.Printf("失效 RSS 已复活 (id=%d, url=%s)", friendRssID, rssURL)
}
