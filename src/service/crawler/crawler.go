package crawlerService

import (
	"blog_api/src/model"
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

const maxHTMLResponseBytes = int64(4 << 20)

var crawlerHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func crawlWebsite(ctx context.Context, rawURL string) model.CrawlResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		log.Printf("[crawler]创建获取 %s 的请求时出错: %v", rawURL, err)
		return model.CrawlResult{Status: "error"}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 -  blog_api_webCrawler")
	resp, err := crawlerHTTPClient.Do(req)
	if err != nil {
		log.Printf("[crawler]获取 URL %s 时出错: %v", rawURL, err)
		return model.CrawlResult{Status: "timeout"}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		redirectLocation := resp.Header.Get("Location")
		absoluteRedirectURL := toAbsoluteURL(resp.Request.URL, redirectLocation)
		log.Printf("[crawler]检测到 %s 重定向到 %s (resolved to %s)", rawURL, redirectLocation, absoluteRedirectURL)
		return model.CrawlResult{Status: "survival", RedirectURL: absoluteRedirectURL}
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[crawler]错误: %s 的状态码非 200: %d", rawURL, resp.StatusCode)
		return model.CrawlResult{Status: "error"}
	}
	limitedBody := &limitedReader{reader: resp.Body, remaining: maxHTMLResponseBytes}
	utf8Reader, err := charset.NewReader(limitedBody, resp.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("[crawler]创建 %s 的字符集解码器时出错: %v", rawURL, err)
		return model.CrawlResult{Status: "error"}
	}
	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		log.Printf("[crawler]解析 %s 的 HTML 时出错: %v", rawURL, err)
		return model.CrawlResult{Status: "error"}
	}
	description := doc.Find("meta[name='description']").AttrOr("content", "")
	iconURL, exists := doc.Find("link[rel='icon']").Attr("href")
	if !exists {
		iconURL, exists = doc.Find("link[rel='apple-touch-icon']").Attr("href")
		if !exists {
			iconURL = doc.Find("link[rel='shortcut icon']").AttrOr("href", "")
		}
	}

	// 查找 RSS feeds
	var atomURLs []string
	var rss2URLs []string
	doc.Find("link[rel='alternate']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			linkType, _ := s.Attr("type")
			absoluteURL := toAbsoluteURL(resp.Request.URL, href)
			if absoluteURL == "" {
				return // Skip if URL is invalid
			}

			switch linkType {
			case "application/atom+xml":
				atomURLs = append(atomURLs, absoluteURL)
			case "application/rss+xml":
				rss2URLs = append(rss2URLs, absoluteURL)
			}
		}
	})

	var rssURLs []string
	if len(atomURLs) > 0 {
		rssURLs = atomURLs
	} else {
		rssURLs = rss2URLs
	}
	if len(rssURLs) == 0 {
		rssURLs = discoverCommonFeedURLs(ctx, resp.Request.URL)
	}

	return model.CrawlResult{
		Description: description,
		IconURL:     iconURL,
		Status:      "survival",
		RssURLs:     rssURLs,
	}
}

// toAbsoluteURL 根据基础 URL 将相对 URL 转换为绝对 URL
func toAbsoluteURL(base *url.URL, href string) string {
	relativeURL, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(relativeURL).String()
}

// discoverCommonFeedURLs 尝试常见 RSS/Atom 地址作为兜底方案
func discoverCommonFeedURLs(ctx context.Context, base *url.URL) []string {
	if base == nil {
		return nil
	}

	root := &url.URL{
		Scheme: base.Scheme,
		Host:   base.Host,
	}
	candidates := []string{
		"/atom.xml",
		"/rss.xml",
		"/feed",
		"/feed.xml",
		"/index.xml",
		"/rss",
		"/atom",
		"/feeds/posts/default?alt=rss",
	}

	found := make([]string, 0, 1)
	for _, candidate := range candidates {
		abs := toAbsoluteURL(root, candidate)
		if abs == "" {
			continue
		}
		feed, err := parseFeedURL(ctx, abs)
		if err == nil && feed != nil {
			found = append(found, abs)
			break
		}
	}
	return found
}
