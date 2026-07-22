package service

import (
	"blog_api/src/model"
	momentRepositories "blog_api/src/repositories/moment"
	"time"

	"gorm.io/gorm"
)

// CreateMoment 创建新的动态
func CreateMoment(db *gorm.DB, req model.CreateMomentRequest) error {
	moment := model.Moment{
		Content:   req.Content,
		Status:    "visible",
		CreatedAt: time.Now().Unix(),
	}
	if req.Tags != nil {
		moment.Tags = *req.Tags
	}
	if req.PinnedOrder != nil {
		moment.PinnedOrder = *req.PinnedOrder
	}
	if req.IsAd != nil {
		moment.IsAd = *req.IsAd
	}
	if req.Extension != nil {
		moment.Extension = *req.Extension
	}
	if req.GuildID != nil {
		moment.GuildID = *req.GuildID
	}
	if req.ChannelID != nil {
		moment.ChannelID = *req.ChannelID
	}
	if req.MessageID != nil {
		moment.MessageID = *req.MessageID
	}
	if req.MessageLink != nil {
		moment.MessageLink = *req.MessageLink
	}

	var media []model.MomentMedia
	for _, m := range req.Media {
		isLocal := 0
		if m.IsLocal != nil {
			isLocal = *m.IsLocal
		}
		media = append(media, model.MomentMedia{
			MediaURL:  m.MediaURL,
			MediaType: m.MediaType,
			IsLocal:   isLocal,
			IsDeleted: 0,
		})
	}

	return momentRepositories.CreateMoment(db, &moment, media)
}

// GetMomentsWithMedia 获取包含媒体文件的动态列表
func GetMomentsWithMedia(db *gorm.DB, page, pageSize int, status string, fingerprintID *int) (*model.QueryMomentsResponse, error) {
	// 查询动态列表和总数
	moments, total, err := momentRepositories.QueryMoments(db, page, pageSize, status)
	if err != nil {
		return nil, err
	}

	// 如果没有动态，直接返回空列表
	if len(moments) == 0 {
		return &model.QueryMomentsResponse{
			Moments: []model.MomentWithMedia{},
			Total:   total,
		}, nil
	}

	// 提取动态 ID 列表
	momentIDs := make([]int, len(moments))
	for i, m := range moments {
		momentIDs[i] = m.ID
	}

	// 获取关联的媒体文件
	mediaList, err := momentRepositories.GetMediaForMoments(db, momentIDs)
	if err != nil {
		return nil, err
	}

	reactionCounts, err := momentRepositories.GetReactionCountsForMoments(db, momentIDs)
	if err != nil {
		return nil, err
	}

	var selectedReactions map[int]string
	if fingerprintID != nil && *fingerprintID > 0 {
		selectedReactions, err = momentRepositories.GetUserReactionsForMoments(db, momentIDs, *fingerprintID)
		if err != nil {
			return nil, err
		}
	}

	// 将媒体文件按 moment_id 分组
	mediaMap := make(map[int][]model.MomentMedia)
	for _, media := range mediaList {
		mediaMap[media.MomentID] = append(mediaMap[media.MomentID], media)
	}

	// 组合动态和媒体文件
	result := make([]model.MomentWithMedia, len(moments))
	for i, m := range moments {
		result[i] = model.MomentWithMedia{
			Moment:    m,
			Media:     mediaMap[m.ID],
			Reactions: reactionCounts[m.ID],
		}
		// 确保 Media 字段不为 nil，方便 JSON 序列化
		if result[i].Media == nil {
			result[i].Media = []model.MomentMedia{}
		}
		if result[i].Reactions == nil {
			result[i].Reactions = map[string]int{}
		}
		if selectedReactions != nil {
			result[i].SelectedReaction = selectedReactions[m.ID]
		}
	}

	return &model.QueryMomentsResponse{
		Moments: result,
		Total:   total,
	}, nil
}
