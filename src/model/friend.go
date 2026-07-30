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
	RejectionReason string `json:"rejection_reason,omitempty" gorm:"column:rejection_reason"`
	Color           string   `json:"color,omitempty" gorm:"column:color"`
	Rss             string   `json:"rss,omitempty" gorm:"column:rss"`
	Tags            []string `json:"tags,omitempty" gorm:"column:tags;type:text;serializer:json"`
}

// TableName sets the insert table name for this struct type.
func (FriendWebsite) TableName() string {
	return "friend_link"
}

// FriendLinkGroup represents a friend link display group.
type FriendLinkGroup struct {
	ID          int    `json:"id" gorm:"column:id;primaryKey"`
	Name        string `json:"name" gorm:"column:name;not null"`
	Description string `json:"description" gorm:"column:description"`
	SortOrder   int    `json:"sort_order" gorm:"column:sort_order;default:0"`
	CreatedAt   int64  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   int64  `json:"updated_at" gorm:"column:updated_at"`
}

// TableName sets the table name for FriendLinkGroup.
func (FriendLinkGroup) TableName() string {
	return "friend_link_group"
}

// FriendLinkGroupMapping maps a friend link to a group.
type FriendLinkGroupMapping struct {
	ID             int `json:"id" gorm:"column:id;primaryKey"`
	FriendLinkID   int `json:"friend_link_id" gorm:"column:friend_link_id;not null;index"`
	FriendLinkGroupID int `json:"friend_link_group_id" gorm:"column:friend_link_group_id;not null;index"`
}

// TableName sets the table name for FriendLinkGroupMapping.
func (FriendLinkGroupMapping) TableName() string {
	return "friend_link_group_mapping"
}

// FriendLinkGroupOutput represents a group in the public friend.json output.
type FriendLinkGroupOutput struct {
	Name  string              `json:"name"`
	Desc  string              `json:"desc"`
	Links []FriendLinkInGroup `json:"links"`
}

// FriendLinkInGroup represents a single friend link inside a group output.
type FriendLinkInGroup struct {
	Name     string   `json:"name"`
	Blog     string   `json:"blog"`
	URL      string   `json:"url"`
	Avatar   string   `json:"avatar"`
	Desc     string   `json:"desc"`
	Color    string   `json:"color"`
	Siteshot string   `json:"siteshot"`
	Rss      string   `json:"rss"`
	Tags     []string `json:"tags"`
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

// RssFetchResult summarizes one completed RSS fetch.
type RssFetchResult struct {
	CheckedItems  int `json:"checked_items"`
	InsertedItems int `json:"inserted_items"`
}

// FriendLinkApplyReq is the public application form payload.
type FriendLinkApplyReq struct {
	Name           string `json:"name"`
	Link           string `json:"link"`
	Avatar         string `json:"avatar"`
	Description    string `json:"description,omitempty"`
	Email          string `json:"email"`
	Snapshot       string `json:"snapshot,omitempty"`
	FriendLinkPage string `json:"friend_link_page,omitempty"`
	Feed           string `json:"feed,omitempty"`
	EnableRss      bool   `json:"enable_rss"`
}

// FriendLinkUpdateApplyReq is the public update application form payload.
type FriendLinkUpdateApplyReq struct {
	OriginalURL    string `json:"original_url"`
	Name           string `json:"name"`
	Link           string `json:"link"`
	Avatar         string `json:"avatar"`
	Description    string `json:"description,omitempty"`
	Email          string `json:"email"`
	Snapshot       string `json:"snapshot,omitempty"`
	FriendLinkPage string `json:"friend_link_page,omitempty"`
	Feed           string `json:"feed,omitempty"`
	EnableRss      bool   `json:"enable_rss"`
}

// FriendLinkSubmission is a public-safe view of an application record.
type FriendLinkSubmission struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	UpdatedAt   int64  `json:"updated_at"`
}
