package model

// StatusData holds the statistical data for the system.
type StatusData struct {
	FriendLinkCount     int                     `json:"friend_link_count"`
	RssCount            int                     `json:"rss_count"`
	RssPostCount        int                     `json:"rss_post_count"`
	FriendLinkStatusPie []FriendLinkStatusCount `json:"friend_link_status_pie"`
	RssPostCountMonthly []RssPostCountMonthly   `json:"rss_post_count_monthly"`
}

// FriendLinkStatusCount holds the count of friend links by status.
type FriendLinkStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// RssPostCountMonthly holds the count of RSS posts by month.
type RssPostCountMonthly struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// SystemStatus represents the overall system status response.
type SystemStatus struct {
	Uptime     string     `json:"uptime"`
	StatusData StatusData `json:"status_data"`
	Time       int64      `json:"time"`
}
