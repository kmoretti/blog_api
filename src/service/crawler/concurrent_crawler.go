package crawlerService

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"context"
	"log"
	"sync"

	"gorm.io/gorm"
)

// CrawlJob 表示一个爬取任务
type CrawlJob struct {
	Link model.FriendWebsite
}

// CrawlJobResult 表示爬取任务的结果
type CrawlJobResult struct {
	Link   model.FriendWebsite
	Result model.CrawlResult
}

type rssParseJob struct {
	Feed model.FriendRss
}

type rssParseJobResult struct {
	feed    model.FriendRss
	fetched fetchedRssFeed
	err     error
}

// RssBatchResult summarizes a completed concurrent RSS scan.
type RssBatchResult struct {
	// ProcessedFeeds is the number of worker outcomes consumed by the writer.
	ProcessedFeeds int

	// SourceFailures is the number of feeds that could not be fetched or parsed.
	SourceFailures int

	// DatabaseFailures is the number of feed results that could not be persisted.
	DatabaseFailures int

	// CheckedItems is the number of items in successfully parsed feeds.
	CheckedItems int

	// InsertedItems is the number of new articles committed to the database.
	InsertedItems int
}

// ImageCheckJob 表示一个图片检查任务
type ImageCheckJob struct {
	Image model.Image
}

// CrawlWebsitesConcurrently crawls links with a bounded worker pool.
// Results are consumed synchronously so database writes retain one owner.
func CrawlWebsitesConcurrently(ctx context.Context, links []model.FriendWebsite, consume func(CrawlJobResult) error) error {
	if len(links) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	concurrency := effectiveConcurrency(len(links))
	log.Printf("[ConcurrentCrawler] 开始并发爬取 %d 个网站，并发数: %d", len(links), concurrency)

	jobs := make(chan CrawlJob, concurrency)
	results := make(chan CrawlJobResult, concurrency)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go crawlWorker(ctx, i, jobs, results, &wg)
	}

	go func() {
		defer close(jobs)
		for _, link := range links {
			select {
			case jobs <- CrawlJob{Link: link}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var consumeErr error
	processed := 0
	for result := range results {
		if consumeErr != nil {
			continue
		}
		if err := consume(result); err != nil {
			consumeErr = err
			cancel()
			continue
		}
		processed++
	}

	log.Printf("[ConcurrentCrawler] 完成并发爬取，共处理 %d 个网站", processed)
	return consumeErr
}

// crawlWorker 是 worker goroutine，从任务通道获取任务并执行爬取
func crawlWorker(ctx context.Context, id int, jobs <-chan CrawlJob, results chan<- CrawlJobResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			log.Printf("[ConcurrentCrawler][Worker %d] 正在爬取: %s", id, job.Link.Link)
			result := CrawlWebsite(ctx, job.Link.Link)
			select {
			case results <- CrawlJobResult{Link: job.Link, Result: result}:
			case <-ctx.Done():
				return
			}
			log.Printf("[ConcurrentCrawler][Worker %d] 完成爬取: %s, 状态: %s", id, job.Link.Link, result.Status)
		}
	}
}

// ParseRssFeedsConcurrently fetches feeds concurrently and persists their
// results through one synchronous database writer. A failure for one feed does
// not prevent later queued feeds from being persisted.
func ParseRssFeedsConcurrently(ctx context.Context, db *gorm.DB, feeds []model.FriendRss) RssBatchResult {
	var summary RssBatchResult
	if len(feeds) == 0 {
		return summary
	}

	activeFeeds := make([]model.FriendRss, 0, len(feeds))
	for _, feed := range feeds {
		if feed.Status == "pause" || feed.IsDied {
			log.Printf("[ConcurrentCrawler] 跳过状态为 %s, is_died=%t 的 RSS 订阅源: %s", feed.Status, feed.IsDied, feed.RssURL)
			continue
		}
		activeFeeds = append(activeFeeds, feed)
	}
	activeCount := len(activeFeeds)
	if activeCount == 0 {
		log.Printf("[ConcurrentCrawler] 没有需要解析的 RSS 订阅源")
		return summary
	}

	concurrency := effectiveConcurrency(activeCount)
	log.Printf("[ConcurrentCrawler] 开始并发解析 %d 个 RSS 订阅源，并发数: %d", activeCount, concurrency)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan rssParseJob, concurrency)
	results := make(chan rssParseJobResult, concurrency)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go rssParseWorker(ctx, i, jobs, results, &wg)
	}

	go func() {
		defer close(jobs)
		for _, feed := range activeFeeds {
			select {
			case jobs <- rssParseJob{Feed: feed}:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for outcome := range results {
		summary.ProcessedFeeds++
		if outcome.err != nil {
			if ctx.Err() != nil {
				continue
			}
			summary.SourceFailures++
			if err := updateRssParseState(db, outcome.feed.ID, false); err != nil {
				summary.DatabaseFailures++
				log.Printf("[ConcurrentCrawler][Writer] 记录 RSS 来源失败状态失败 (id=%d): %v", outcome.feed.ID, err)
			}
			log.Printf("[ConcurrentCrawler][Writer] RSS 来源解析失败 (id=%d, url=%s): %v", outcome.feed.ID, outcome.feed.RssURL, outcome.err)
			continue
		}

		summary.CheckedItems += outcome.fetched.checkedItems
		result, err := persistFetchedRssFeed(db, outcome.fetched)
		if err != nil {
			summary.DatabaseFailures++
			log.Printf("[ConcurrentCrawler][Writer] RSS 入库失败 (id=%d, url=%s): %v", outcome.feed.ID, outcome.feed.RssURL, err)
			continue
		}
		summary.InsertedItems += result.InsertedItems
		log.Printf("RSS %s 共检查 %d 篇文章，新增 %d 篇", outcome.feed.RssURL, result.CheckedItems, result.InsertedItems)
	}

	log.Printf(
		"[ConcurrentCrawler] 完成并发解析：处理 %d 个 RSS，来源失败 %d 个，数据库失败 %d 个，共检查 %d 篇文章，新增 %d 篇",
		summary.ProcessedFeeds,
		summary.SourceFailures,
		summary.DatabaseFailures,
		summary.CheckedItems,
		summary.InsertedItems,
	)
	return summary
}

func rssParseWorker(
	ctx context.Context,
	id int,
	jobs <-chan rssParseJob,
	results chan<- rssParseJobResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			log.Printf("[ConcurrentCrawler][Worker %d] 正在抓取 RSS: %s", id, job.Feed.RssURL)
			fetched, err := fetchRssFeed(ctx, job.Feed)
			outcome := rssParseJobResult{feed: job.Feed, fetched: fetched, err: err}
			select {
			case results <- outcome:
				log.Printf("[ConcurrentCrawler][Worker %d] 完成抓取 RSS: %s", id, job.Feed.RssURL)
			case <-ctx.Done():
				return
			}
		}
	}
}

// CheckImagesConcurrently 并发检查图片
func CheckImagesConcurrently(images []model.Image, checkFunc func(image model.Image)) {
	if len(images) == 0 {
		return
	}

	concurrency := effectiveConcurrency(len(images))
	log.Printf("[ConcurrentCrawler] 开始并发检查 %d 张图片，并发数: %d", len(images), concurrency)

	jobs := make(chan ImageCheckJob, concurrency)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go imageCheckWorker(i, jobs, checkFunc, &wg)
	}

	for _, img := range images {
		jobs <- ImageCheckJob{Image: img}
	}
	close(jobs)

	wg.Wait()
	log.Printf("[ConcurrentCrawler] 完成并发检查 %d 张图片", len(images))
}

func imageCheckWorker(id int, jobs <-chan ImageCheckJob, checkFunc func(image model.Image), wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		log.Printf("[ConcurrentCrawler][Worker %d] 正在检查图片: %s", id, job.Image.URL)
		checkFunc(job.Image)
		log.Printf("[ConcurrentCrawler][Worker %d] 完成检查图片: %s", id, job.Image.URL)
	}
}

func effectiveConcurrency(total int) int {
	concurrency := config.GetConfig().Crawler.Concurrency
	if concurrency <= 0 {
		concurrency = 5 // 默认并发数
	}
	if total < concurrency {
		return total
	}
	return concurrency
}
