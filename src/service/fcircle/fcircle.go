package fcircle

import (
	"blog_api/src/model"
	friendsRepositories "blog_api/src/repositories/friend"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	filename          = "fcircle.json"
	friendJSONName    = "friend.json"
	regenerateDelay   = 5 * time.Second
	maxSurvivalFriend = 10000
)

// Service generates and serves fcircle.json for external friend-circle consumers.
type Service struct {
	db      *gorm.DB
	dataDir string
	mu      sync.Mutex
	timer   *time.Timer
	pending bool
}

var (
	defaultSvc *Service
	initMu     sync.Mutex
)

// NewService creates a new fcircle generator service.
func NewService(db *gorm.DB, dataDir string) *Service {
	return &Service{
		db:      db,
		dataDir: dataDir,
	}
}

// Init initializes the package-level default service and generates the first file.
func Init(db *gorm.DB, dataDir string) error {
	if db == nil {
		return errors.New("database connection is nil")
	}
	initMu.Lock()
	defer initMu.Unlock()
	defaultSvc = NewService(db, dataDir)
	return defaultSvc.Generate()
}

// ScheduleRegenerate schedules a debounced regeneration of fcircle.json.
// It is safe to call from any goroutine and will not block the caller.
func ScheduleRegenerate() {
	initMu.Lock()
	s := defaultSvc
	initMu.Unlock()
	if s != nil {
		s.ScheduleRegenerate()
	}
}

// Handler returns a gin handler that serves the generated fcircle.json file.
func Handler() gin.HandlerFunc {
	initMu.Lock()
	s := defaultSvc
	initMu.Unlock()
	if s == nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, model.NewErrorResponse(503, "fcircle service not initialized"))
		}
	}
	return s.Handler()
}

// Generate queries all survival friend links and writes both fcircle.json and
// friend.json atomically.
func (s *Service) Generate() error {
	opts := model.FriendLinkQueryOptions{
		Status: "survival",
		Limit:  maxSurvivalFriend,
	}
	resp, err := friendsRepositories.QueryFriendLinks(s.db, opts)
	if err != nil {
		return err
	}

	friends := make([][4]string, 0, len(resp.Links))
	for _, link := range resp.Links {
		friends = append(friends, [4]string{
			link.Name,
			link.Link,
			link.FriendLinkPage,
			link.Avatar,
		})
	}

	fcircleData := map[string]interface{}{
		"friends": friends,
	}

	if err := writeJSONFile(s.dataDir, filename, fcircleData); err != nil {
		return err
	}

	friendJSONData, err := buildFriendJSON(s.db, resp.Links)
	if err != nil {
		return err
	}
	return writeJSONFile(s.dataDir, friendJSONName, friendJSONData)
}

// GenerateFriendJSON builds the grouped friend.json data in memory.
// It is exported so the HTTP handler can serve it without touching the disk.
func GenerateFriendJSON(db *gorm.DB) (map[string]interface{}, error) {
	opts := model.FriendLinkQueryOptions{
		Status: "survival",
		Limit:  maxSurvivalFriend,
	}
	resp, err := friendsRepositories.QueryFriendLinks(db, opts)
	if err != nil {
		return nil, err
	}
	return buildFriendJSON(db, resp.Links)
}

func buildFriendJSON(db *gorm.DB, links []model.FriendWebsite) (map[string]interface{}, error) {
	groups, err := friendsRepositories.ListFriendLinkGroups(db)
	if err != nil {
		return nil, err
	}

	defaultID, err := friendsRepositories.EnsureDefaultFriendLinkGroup(db)
	if err != nil {
		return nil, err
	}

	groupIndex := make(map[int]int, len(groups))
	outputs := make([]model.FriendLinkGroupOutput, 0, len(groups)+1)
	for _, g := range groups {
		groupIndex[g.ID] = len(outputs)
		outputs = append(outputs, model.FriendLinkGroupOutput{
			Name:  g.Name,
			Desc:  g.Description,
			Links: make([]model.FriendLinkInGroup, 0),
		})
	}

	// Ensure the default group is present even if it was manually deleted.
	if _, ok := groupIndex[defaultID]; !ok {
		groupIndex[defaultID] = len(outputs)
		outputs = append(outputs, model.FriendLinkGroupOutput{
			Name:  "网上邻居",
			Desc:  "",
			Links: make([]model.FriendLinkInGroup, 0),
		})
	}

	for _, link := range links {
		ids, err := friendsRepositories.GetFriendLinkGroupIDs(db, link.ID)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			ids = []int{defaultID}
		}

		item := model.FriendLinkInGroup{
			Name:     link.Name,
			Blog:     link.Name,
			URL:      link.Link,
			Avatar:   link.Avatar,
			Desc:     link.Info,
			Color:    link.Color,
			Siteshot: link.Snapshot,
			Rss:      firstNonEmpty(link.Rss, link.Feed),
			Tags:     link.Tags,
		}

		seen := make(map[int]struct{}, len(ids))
		for _, gid := range ids {
			if _, ok := seen[gid]; ok {
				continue
			}
			seen[gid] = struct{}{}
			idx, ok := groupIndex[gid]
			if !ok {
				// Group no longer exists; fall back to default group.
				idx = groupIndex[defaultID]
			}
			outputs[idx].Links = append(outputs[idx].Links, item)
		}
	}

	// Remove groups that ended up empty.
	filtered := make([]model.FriendLinkGroupOutput, 0, len(outputs))
	for _, g := range outputs {
		if len(g.Links) > 0 {
			filtered = append(filtered, g)
		}
	}

	return map[string]interface{}{
		"linkGroups": filtered,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func writeJSONFile(dataDir, name string, data interface{}) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	tmpPath := filepath.Join(dataDir, "."+name+".tmp")
	finalPath := filepath.Join(dataDir, name)

	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, finalPath)
}

// ScheduleRegenerate schedules a debounced regeneration. Multiple calls within
// regenerateDelay are collapsed into a single write to reduce disk pressure.
func (s *Service) ScheduleRegenerate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pending = true
	if s.timer != nil {
		s.timer.Stop()
	}

	s.timer = time.AfterFunc(regenerateDelay, func() {
		s.mu.Lock()
		s.pending = false
		s.mu.Unlock()

		if err := s.Generate(); err != nil {
			log.Printf("[fcircle] 重新生成 %s 失败: %v", filename, err)
		} else {
			log.Printf("[fcircle] 已重新生成 %s", filename)
		}
	})
}

// Handler returns a gin handler that serves fcircle.json. If the file does not
// exist yet, it attempts to generate it on the fly.
func (s *Service) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := filepath.Join(s.dataDir, filename)

		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			if os.IsNotExist(err) {
				if genErr := s.Generate(); genErr != nil {
					log.Printf("[fcircle] 首次生成 %s 失败: %v", filename, genErr)
					c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to generate fcircle.json"))
					return
				}
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.File(path)
				return
			}
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(500, "failed to access fcircle.json"))
			return
		}

		c.Header("Content-Type", "application/json; charset=utf-8")
		c.File(path)
	}
}
