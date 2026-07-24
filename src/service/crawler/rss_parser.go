package crawlerService

import (
	"blog_api/src/config"
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

const (
	maxRssParseFailures = 4
	maxRSSResponseBytes = int64(8 << 20)
)

var rssHTTPClient = &http.Client{}

func parseFeedURL(ctx context.Context, rawURL string) (*gofeed.Feed, error) {
	timeoutSeconds := config.GetConfig().Crawler.RssTimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create RSS request: %w", err)
	}
	req.Header.Set("User-Agent", "blog_api RSS crawler")
	resp, err := rssHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch RSS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch RSS: unexpected HTTP status %d", resp.StatusCode)
	}

	feed, err := gofeed.NewParser().Parse(&limitedReader{reader: resp.Body, remaining: maxRSSResponseBytes})
	if err != nil {
		return nil, fmt.Errorf("parse RSS: %w", err)
	}
	return feed, nil
}

// ParseRssFeed parses an RSS feed and saves the articles to the database.
func ParseRssFeed(db *gorm.DB, friendRssID int, rssURL string) {
	feed, err := parseFeedURL(context.Background(), rssURL)
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
				continue
			}
		}

		publishedUnix := publishedTime.Unix()
		if publishedUnix < 0 {
			publishedUnix = 0
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
			Time:        publishedUnix,
		}

		err := friendsRepositories.InsertRssPost(db, post)
		if err != nil {
			log.Printf("插入文章 '%s' 时出错: %v", item.Title, err)
		}
	}

	log.Printf("RSS %s 共检查 %d 篇文章", rssURL, len(feed.Items))
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
	feed, err := parseFeedURL(context.Background(), rssURL)
	if err != nil {
		log.Printf("解析 RSS feed %s 时出错: %v", rssURL, err)
		return "", err
	}
	return feed.Title, nil
}

// CheckAndReviveRssFeed probes a died RSS feed and revives it on success.
func CheckAndReviveRssFeed(db *gorm.DB, friendRssID int, rssURL string) {
	if _, err := parseFeedURL(context.Background(), rssURL); err != nil {
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
