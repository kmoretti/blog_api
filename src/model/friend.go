package model

// FriendWebsite 单个友链站点
type FriendWebsite struct {
	ID              int    `json:"id,omitempty" gorm:"column:id;primaryKey"`
	Name            string `json:"name" gorm:"column:website_name"`
	Link            string `json:"link" gorm:"column:website_url"`
	Avatar          string `json:"avatar" gorm:"column:website_icon_url"`
	Info            string `json:"description" gorm:"column:description"`
	Email           string `json:"email,omitempty" gorm:"column:email"`
	Times           int    `json:"times,omitempty" gorm:"column:times"`
	Status          string `json:"status,omitempty" gorm:"column:status"`
	IsDied          bool   `json:"is_died,omitempty" gorm:"column:is_died"`
	EnableRss       bool   `json:"enable_rss,omitempty" gorm:"column:enable_rss"`
	SkipHealthCheck bool   `json:"skip_health_check,omitempty" gorm:"column:skip_health_check"`
	UpdatedAt       int64  `json:"updated_at,omitempty" gorm:"column:updated_at"`
	Snapshot        string `json:"snapshot,omitempty" gorm:"column:snapshot"`
	FriendLinkPage  string `json:"friend_link_page,omitempty" gorm:"column:friend_link_page"`
	Feed            string `json:"feed,omitempty" gorm:"column:feed"`
}

// TableName sets the insert table name for this struct type.
func (FriendWebsite) TableName() string {
	return "friend_link"
}

// FriendLinkQueryOptions defines the options for querying friend links.
type FriendLinkQueryOptions struct {
	Status          string
	Statuses        []string
	Email           string
	IsDied          *bool
	SkipHealthCheck *bool
	NotIn           bool
	Search          string
	Offset          int
	Limit           int
	Count           bool
}

// QueryFriendLinksResponse defines the response for the unified friend link query.
type QueryFriendLinksResponse struct {
	Links []FriendWebsite
	Count int64
}

// FriendRss maps to the friend_rss table.
type FriendRss struct {
	ID           int    `json:"id" gorm:"column:id;primaryKey"`
	FriendLinkID int    `json:"friend_link_id" gorm:"column:friend_link_id"`
	Name         string `json:"name" gorm:"column:name"`
	RssURL       string `json:"rss_url" gorm:"column:rss_url"`
	Times        int    `json:"times" gorm:"column:times"`
	Status       string `json:"status" gorm:"column:status"`
	IsDied       bool   `json:"is_died" gorm:"column:is_died"`
	UpdatedAt    int64  `json:"updated_at" gorm:"column:updated_at"`
}

// RssPost represents an article from an RSS feed.
type RssPost struct {
	ID          int    `json:"id" gorm:"column:id;primaryKey"`
	RssID       int    `json:"rss_id" gorm:"column:rss_id"`
	Title       string `json:"title" gorm:"column:title"`
	Link        string `json:"link" gorm:"column:link"`
	Description string `json:"description" gorm:"column:description"`
	Author      string `json:"author" gorm:"column:author"`
	Time        int64  `json:"time" gorm:"column:time"`
}

// TableName sets the table name for FriendRss.
func (FriendRss) TableName() string {
	return "friend_rss"
}

// TableName sets the table name for RssPost.
func (RssPost) TableName() string {
	return "friend_rss_post"
}

// FriendRssQueryOptions defines the options for querying friend RSS feeds.
type FriendRssQueryOptions struct {
	FriendLinkID int    // Filter by friend link ID
	Status       string // Filter by status
	IsDied       *bool  // Filter by is_died status
	Page         int    // Page number for pagination
	PageSize     int    // Number of items per page
}

// QueryFriendRssResponse defines the response for the unified friend RSS query.
type QueryFriendRssResponse struct {
	Feeds []FriendRss `json:"feeds"`
	Total int64       `json:"total"`
}
