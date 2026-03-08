package cmd

import (
	"blog_api/src/config"
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	crawlerService "blog_api/src/service/crawler"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

func scheduleFromNextMidnight(jobName string, interval time.Duration, job func()) {
	go func() {
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		firstRunAt := nextMidnight.Add(interval)
		initialDelay := time.Until(firstRunAt)

		log.Printf("[Cron] %s 将于 %s 首次执行（下一天 0 点后 + %s），之后每 %s 执行一次", jobName, firstRunAt.Format(time.RFC3339), interval, interval)

		timer := time.NewTimer(initialDelay)
		defer timer.Stop()

		<-timer.C
		job()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			job()
		}
	}()
}

// RunFriendLinkCrawlerJob 执行友链爬取并发现 RSS 订阅源（并发模式）
func RunFriendLinkCrawlerJob(db *gorm.DB) {
	log.Println("[Cron] 正在运行友链爬取任务（并发模式）...")
	isDied := false
	opts := model.FriendLinkQueryOptions{
		Statuses: []string{"ignored"},
		NotIn:    true,
		IsDied:   &isDied,
	}
	resp, err := friendsRepositories.QueryFriendLinks(db, opts)
	if err != nil {
		log.Printf("[Cron] 获取全部友链失败： %v", err)
		return
	}
	links := resp.Links

	if len(links) == 0 {
		log.Println("[Cron] 没有需要爬取的友链")
		return
	}

	// 使用并发爬虫
	results := crawlerService.CrawlWebsitesConcurrently(links)

	// 处理爬取结果
	for _, crawlResult := range results {
		link := crawlResult.Link
		result := crawlResult.Result

		err := friendsRepositories.UpdateFriendLink(db, link, result)
		if err != nil {
			log.Printf("[Cron] 在 cron 任务中更新友链 %s 失败: %v", link.Name, err)
		}
		// 更新友链后，发现并插入 RSS 订阅源
		if link.EnableRss && len(result.RssURLs) > 0 {
			for _, rssURL := range result.RssURLs {
				name, err := crawlerService.GetRssTitle(rssURL)
				if err != nil {
					log.Printf("[Cron] 获取 RSS 标题失败 %s: %v", rssURL, err)
					continue
				}
				_, err = friendsRepositories.CreateFriendRssFeeds(db, link.ID, rssURL, name)
				if err != nil {
					log.Printf("[Cron] 在 cron 任务中为 %s 插入 RSS 订阅源失败: %v", link.Name, err)
				}
			}
		}
	}
	log.Println("[Cron] 友链爬取任务完成")
}

// RunDiedFriendLinkCheckJob 执行失效友链的检查（并发模式）
func RunDiedFriendLinkCheckJob(db *gorm.DB) {
	log.Println("[Cron] 正在运行失效友链检查任务（并发模式）...")
	isDied := true
	opts := model.FriendLinkQueryOptions{
		IsDied: &isDied,
	}
	resp, err := friendsRepositories.QueryFriendLinks(db, opts)
	if err != nil {
		log.Printf("[Cron] 获取全部 died 友链失败： %v", err)
		return
	}
	links := resp.Links

	if len(links) == 0 {
		log.Println("[Cron] 没有需要检查的失效友链")
		return
	}

	// 使用并发爬虫
	results := crawlerService.CrawlWebsitesConcurrently(links)

	// 处理爬取结果
	for _, crawlResult := range results {
		link := crawlResult.Link
		result := crawlResult.Result
		// 如果链接仍然有效，状态将更新为"存活"并重置计数
		err := friendsRepositories.UpdateFriendLink(db, link, result)
		if err != nil {
			log.Printf("[Cron] 在 cron 任务中更新失效友链 %s 失败: %v", link.Name, err)
		}
	}
	log.Println("[Cron] 失效友链检查任务完成")
}

// RunDiedRssCheckJob 执行失效 RSS 的探活检查（低频）
func RunDiedRssCheckJob(db *gorm.DB) {
	log.Println("[Cron] 正在运行失效 RSS 探活任务...")
	isDied := true
	opts := model.FriendRssQueryOptions{
		IsDied: &isDied,
	}
	resp, err := friendsRepositories.QueryFriendRss(db, opts)
	if err != nil {
		log.Printf("[Cron] 获取失效 RSS 失败: %v", err)
		return
	}
	rssFeeds := resp.Feeds

	if len(rssFeeds) == 0 {
		log.Println("[Cron] 没有需要探活的失效 RSS")
		return
	}

	for _, feed := range rssFeeds {
		if feed.Status == "pause" {
			continue
		}
		crawlerService.CheckAndReviveRssFeed(db, feed.ID, feed.RssURL)
	}

	log.Println("[Cron] 失效 RSS 探活任务完成")
}

// RunRssParserJob 获取所有 RSS 订阅源并解析它们（并发模式）
func RunRssParserJob(db *gorm.DB) {
	log.Println("[Cron] 正在运行 RSS 解析任务（并发模式）...")
	opts := model.FriendRssQueryOptions{Status: "valid"}
	resp, err := friendsRepositories.QueryFriendRss(db, opts)
	if err != nil {
		log.Printf("[Cron] 获取所有 RSS 订阅源失败: %v", err)
		return
	}
	rssFeeds := resp.Feeds

	if len(rssFeeds) == 0 {
		log.Println("[Cron] 没有需要解析的 RSS 订阅源")
		return
	}

	// 使用并发解析
	crawlerService.ParseRssFeedsConcurrently(rssFeeds, func(friendRssID int, rssURL string) {
		crawlerService.ParseRssFeed(db, friendRssID, rssURL)
	})
	log.Println("[Cron] RSS 解析任务完成")
}

func scheduleDiedCheckEvery48h(db *gorm.DB) {
	scheduleFromNextMidnight("失效检查（友链+RSS）", 48*time.Hour, func() {
		RunDiedFriendLinkCheckJob(db)
		RunDiedRssCheckJob(db)
	})
}

func scheduleImageCheckEvery48h(db *gorm.DB) {
	scheduleFromNextMidnight("图片资源检查", 48*time.Hour, func() {
		crawlerService.CheckImagesHealth(db)
	})
}

// StartCronJobs 初始化并启动 cron 任务
func StartCronJobs(db *gorm.DB) {
	c := cron.New()

	// 安排友链爬取任务每 6 小时运行一次
	c.AddFunc("0 */6 * * *", func() {
		RunFriendLinkCrawlerJob(db)
	})

	// 安排慢检查任务从下一天 0 点开始，每 48 小时运行一次
	scheduleDiedCheckEvery48h(db)
	scheduleImageCheckEvery48h(db)

	// 安排 RSS 解析任务每 3 小时运行一次
	c.AddFunc("0 */3 * * *", func() {
		RunRssParserJob(db)
	})
	// 如果配置了启动时扫描，则立即运行一次任务
	if config.GetConfig().CronScanOnStartup {
		go func() {
			log.Println("[Cron] 调度启动时扫描任务")
			RunFriendLinkCrawlerJob(db)
			RunRssParserJob(db)
		}()
	} else {
		log.Println("[Cron] 根据 CRON_SCAN_ON_STARTUP 设置跳过初始扫描")
	}

	log.Println("[Cron] 正在启动 cron 任务...")
	c.Start()
}
