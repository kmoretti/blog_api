package crawlerService

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"context"
	"log"
	"sync"
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

// RssParseJob 表示一个 RSS 解析任务
type RssParseJob struct {
	FriendRssID int
	RssURL      string
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

// ParseRssFeedsConcurrently 并发解析多个 RSS 订阅源
func ParseRssFeedsConcurrently(feeds []model.FriendRss, parseFunc func(friendRssID int, rssURL string)) {
	if len(feeds) == 0 {
		return
	}

	activeCount := 0
	for _, feed := range feeds {
		if feed.Status == "pause" || feed.IsDied {
			log.Printf("[ConcurrentCrawler] 跳过状态为 %s, is_died=%t 的 RSS 订阅源: %s", feed.Status, feed.IsDied, feed.RssURL)
			continue
		}
		activeCount++
	}
	if activeCount == 0 {
		log.Printf("[ConcurrentCrawler] 没有需要解析的 RSS 订阅源")
		return
	}

	concurrency := effectiveConcurrency(activeCount)
	log.Printf("[ConcurrentCrawler] 开始并发解析 %d 个 RSS 订阅源，并发数: %d", activeCount, concurrency)

	// 创建任务通道
	jobs := make(chan RssParseJob, concurrency)

	// 启动 worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go rssParseWorker(i, jobs, parseFunc, &wg)
	}

	// 发送任务到任务通道
	for _, feed := range feeds {
		if feed.Status == "pause" || feed.IsDied {
			continue
		}
		jobs <- RssParseJob{
			FriendRssID: feed.ID,
			RssURL:      feed.RssURL,
		}
	}
	close(jobs)

	// 等待所有 worker 完成
	wg.Wait()

	log.Printf("[ConcurrentCrawler] 完成并发解析 %d 个 RSS 订阅源", activeCount)
}

// rssParseWorker 是 RSS 解析的 worker goroutine
func rssParseWorker(id int, jobs <-chan RssParseJob, parseFunc func(friendRssID int, rssURL string), wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		log.Printf("[ConcurrentCrawler][Worker %d] 正在解析 RSS: %s", id, job.RssURL)
		parseFunc(job.FriendRssID, job.RssURL)
		log.Printf("[ConcurrentCrawler][Worker %d] 完成解析 RSS: %s", id, job.RssURL)
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
