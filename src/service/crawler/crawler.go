package crawlerService

import (
	"blog_api/src/model"
	"log"
	"net/http"
	"net/url"
	"time"

	"bytes"
	"io"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

// CrawlWebsite 获取并解析网站以提取 SEO 信息
func CrawlWebsite(url string) model.CrawlResult {
	client := &http.Client{
		Timeout: 10 * time.Second, // 设置超时以防止挂起
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[crawler]创建获取 %s 的请求时出错: %v", url, err)
		return model.CrawlResult{Status: "error"}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 -  blog_api_webCrawler")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[crawler]获取 URL %s 时出错: %v", url, err)
		return model.CrawlResult{Status: "timeout"}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		redirectLocation := resp.Header.Get("Location")
		absoluteRedirectURL := toAbsoluteURL(resp.Request.URL, redirectLocation)
		log.Printf("[crawler]检测到 %s 重定向到 %s (resolved to %s)", url, redirectLocation, absoluteRedirectURL)
		return model.CrawlResult{Status: "survival", RedirectURL: absoluteRedirectURL}
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[crawler]错误: %s 的状态码非 200: %d", url, resp.StatusCode)
		return model.CrawlResult{Status: "error"}
	}

	// 读取响应体内容
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[crawler]读取 %s 的响应体时出错: %v", url, err)
		return model.CrawlResult{Status: "error"}
	}
	// 重置响应体以便后续读取
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 确定编码
	e, name, _ := charset.DetermineEncoding(bodyBytes, resp.Header.Get("Content-Type"))
	log.Printf("[crawler]确定 %s 的编码为: %s", url, name)

	// 使用检测到的编码创建读取器
	utf8Reader := e.NewDecoder().Reader(bytes.NewBuffer(bodyBytes))

	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		log.Printf("[crawler]解析 %s 的 HTML 时出错: %v", url, err)
		return model.CrawlResult{Status: "error"}
	}

	// 查找描述
	description := doc.Find("meta[name='description']").AttrOr("content", "")

	// 查找网站图标
	iconURL, exists := doc.Find("link[rel='icon']").Attr("href")
	if !exists {
		// 兼容 apple-touch-icon 或 shortcut icon
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
		rssURLs = discoverCommonFeedURLs(resp.Request.URL)
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
func discoverCommonFeedURLs(base *url.URL) []string {
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
		if isValidFeedURL(abs) {
			found = append(found, abs)
			break
		}
	}
	return found
}

func isValidFeedURL(feedURL string) bool {
	fp := newRssParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil || feed == nil {
		return false
	}
	return true
}
