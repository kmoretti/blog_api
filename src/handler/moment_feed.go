package handler

import (
	"blog_api/src/config"
	"blog_api/src/model"
	"blog_api/src/service"
	"encoding/xml"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const momentFeedPageSize = 50

// atomFeed represents an Atom 1.0 feed.
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Xmlns   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Author  *atomAuthor `xml:"author"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Rel  string `xml:"rel,attr,omitempty"`
	Href string `xml:"href,attr"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	Title      string         `xml:"title"`
	Link       atomLink       `xml:"link"`
	ID         string         `xml:"id"`
	Updated    string         `xml:"updated"`
	Published  string         `xml:"published"`
	Author     *atomAuthor    `xml:"author"`
	Content    *atomContent   `xml:"content"`
	Categories []atomCategory `xml:"category"`
}

type atomContent struct {
	Type string `xml:"type,attr"`
	Body string `xml:",chardata"`
}

type atomCategory struct {
	Term string `xml:"term,attr"`
}

// GetMomentFeed handles GET /api/public/moments/feed and returns an Atom 1.0 feed.
func (h *MomentHandler) GetMomentFeed(c *gin.Context) {
	cfg := config.GetConfig()
	siteURL := strings.TrimRight(cfg.Site.URL, "/")
	if siteURL == "" {
		siteURL = "https://" + c.Request.Host
	}
	feedURL := siteURL + "/api/public/moments/feed"
	siteName := cfg.Site.Name
	if siteName == "" {
		siteName = "朋友圈"
	}

	resp, err := service.GetMomentsWithMedia(h.DB, 1, momentFeedPageSize, "visible", nil)
	if err != nil {
		c.Header("Content-Type", "application/atom+xml; charset=utf-8")
		c.String(http.StatusInternalServerError, atomErrorFeed(siteName, feedURL, siteURL, "动态列表加载失败"))
		return
	}

	feedUpdated := time.Now().UTC().Format(time.RFC3339)
	entries := make([]atomEntry, 0, len(resp.Moments))
	for _, moment := range resp.Moments {
		permalink := siteURL + "/memo/" + strconv.Itoa(moment.ID)
		updatedAt := maxUnix(moment.UpdatedAt, moment.CreatedAt)
		publishedAt := time.Unix(moment.CreatedAt, 0).UTC().Format(time.RFC3339)
		updatedTime := time.Unix(updatedAt, 0).UTC().Format(time.RFC3339)

		contentHTML := buildMomentContentHTML(moment.Content, moment.Media)
		entry := atomEntry{
			Title:     firstLine(moment.Content, moment.ID),
			Link:      atomLink{Href: permalink},
			ID:        permalink,
			Updated:   updatedTime,
			Published: publishedAt,
			Author:    &atomAuthor{Name: siteName},
			Content: &atomContent{
				Type: "html",
				Body: contentHTML,
			},
			Categories: parseTags(moment.Tags),
		}
		entries = append(entries, entry)

		if updatedTime > feedUpdated {
			feedUpdated = updatedTime
		}
	}

	feed := atomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: siteName,
		Link: []atomLink{
			{Rel: "self", Href: feedURL, Type: "application/atom+xml"},
			{Rel: "alternate", Href: siteURL},
		},
		ID:      siteURL,
		Updated: feedUpdated,
		Author:  &atomAuthor{Name: siteName},
		Entries: entries,
	}

	output, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		c.Header("Content-Type", "application/atom+xml; charset=utf-8")
		c.String(http.StatusInternalServerError, atomErrorFeed(siteName, feedURL, siteURL, "Feed 生成失败"))
		return
	}

	c.Header("Content-Type", "application/atom+xml; charset=utf-8")
	c.String(http.StatusOK, xml.Header+string(output))
}

func buildMomentContentHTML(content string, media []model.MomentMedia) string {
	var b strings.Builder
	if trimmed := strings.TrimSpace(content); trimmed != "" {
		b.WriteString("<p>")
		escaped := html.EscapeString(trimmed)
		escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
		b.WriteString(escaped)
		b.WriteString("</p>")
	}
	for _, m := range media {
		url := m.MediaURL
		if url == "" {
			continue
		}
		if strings.HasPrefix(m.MediaType, "video/") {
			b.WriteString("<p><video controls src=\"")
			b.WriteString(html.EscapeString(url))
			b.WriteString("\"></video></p>")
		} else {
			b.WriteString("<p><img src=\"")
			b.WriteString(html.EscapeString(url))
			b.WriteString("\" alt=\"\"></p>")
		}
	}
	return b.String()
}

func firstLine(content string, id int) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 80 {
				line = line[:80] + "…"
			}
			return line
		}
	}
	return "动态 #" + strconv.Itoa(id)
}

func parseTags(tags string) []atomCategory {
	if tags == "" {
		return nil
	}
	var categories []atomCategory
	for _, tag := range strings.Split(tags, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			categories = append(categories, atomCategory{Term: tag})
		}
	}
	return categories
}

func maxUnix(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func atomErrorFeed(siteName, feedURL, siteURL, message string) string {
	feed := atomFeed{
		Xmlns: "http://www.w3.org/2005/Atom",
		Title: siteName,
		Link: []atomLink{
			{Rel: "self", Href: feedURL, Type: "application/atom+xml"},
			{Rel: "alternate", Href: siteURL},
		},
		ID:      siteURL,
		Updated: time.Now().UTC().Format(time.RFC3339),
		Author:  &atomAuthor{Name: siteName},
		Entries: []atomEntry{
			{
				Title:   message,
				ID:      siteURL + "#error",
				Updated: time.Now().UTC().Format(time.RFC3339),
				Content: &atomContent{Type: "text", Body: message},
			},
		},
	}
	output, _ := xml.MarshalIndent(feed, "", "  ")
	return xml.Header + string(output)
}
