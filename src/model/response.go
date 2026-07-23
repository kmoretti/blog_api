package model

import "time"

// ApiResponse 统一API响应结构
type ApiResponse struct {
	Code    int         `json:"code"`    // HTTP状态码
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Items    interface{} `json:"items"`     // 数据列表
	Total    int         `json:"total"`     // 总数量
	Page     int         `json:"page"`      // 当前页码
	PageSize int         `json:"page_size"` // 每页数量
}

// FriendLinkDTO 友链数据传输对象（不包含敏感字段times）
type FriendLinkDTO struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Link            string `json:"link"`
	Avatar          string `json:"avatar"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	Email           string `json:"email,omitempty"`
	Times           int    `json:"times,omitempty"`
	EnableRss       bool   `json:"enable_rss"`
	IsDied          bool   `json:"is_died,omitempty"`
	SkipHealthCheck *bool  `json:"skip_health_check,omitempty"`
	UpdatedAt       int64  `json:"updated_at"`
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) ApiResponse {
	return ApiResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}
}

// MomentWithMedia represents a moment with its associated media files.
type MomentWithMedia struct {
	Moment
	Media            []MomentMedia  `json:"media"`
	Reactions        map[string]int `json:"reactions"`
	SelectedReaction string         `json:"selected_reaction,omitempty"`
}

// PublicMomentWithMedia represents a moment for public APIs (excludes internal IDs).
type PublicMomentWithMedia struct {
	ID               int            `json:"id"`
	Content          string         `json:"content"`
	Status           string         `json:"status"`
	MessageLink      string         `json:"message_link,omitempty"`
	CreatedAt        int64          `json:"created_at"`
	UpdatedAt        int64          `json:"updated_at"`
	Media            []MomentMedia  `json:"media"`
	Reactions        map[string]int `json:"reactions"`
	SelectedReaction string         `json:"selected_reaction,omitempty"`
}

// QueryMomentsResponse defines the response for querying moments.
type QueryMomentsResponse struct {
	Moments []MomentWithMedia `json:"moments"`
	Total   int64             `json:"total"`
}

// QueryMediaResponse defines the response for querying media.
type QueryMediaResponse struct {
	Media []MomentMedia `json:"media"`
	Total int64         `json:"total"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) ApiResponse {
	return ApiResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// FileInfo represents a file or directory in the resource list.
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}
