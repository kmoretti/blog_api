package handler

import (
	"blog_api/src/model"
	imageRepositories "blog_api/src/repositories/image"
	"errors"
	"log"
	"net/http"
	"strconv"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// ImagePublicHandler 处理与图片相关的公开API请求
type ImagePublicHandler struct {
	db *gorm.DB
}

// NewImagePublicHandler 创建一个新的 ImagePublicHandler 实例
func NewImagePublicHandler(db *gorm.DB) *ImagePublicHandler {
	return &ImagePublicHandler{db: db}
}

func (h *ImagePublicHandler) GetImage(c *gin.Context) {
	idStr := c.Param("id")
	var image *model.Image
	var err error

	if idStr == "" || idStr == "/" {
		image, err = imageRepositories.GetRandomImage(h.db)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "没有可用的图片"})
			} else {
				log.Printf("[handler][image_public][ERR] 获取随机图片失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			}
			return
		}
	} else {
		id, errConv := strconv.Atoi(idStr)
		if errConv != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的图片ID"})
			return
		}

		image, err = imageRepositories.GetImageByID(h.db, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "图片未找到"})
			} else {
				log.Printf("[handler][image_public][ERR] 查询图片失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "内部服务器错误"})
			}
			return
		}
	}

	queryType := c.DefaultQuery("type", "image")
	if queryType == "metadata" {
		c.JSON(http.StatusOK, image)
	} else {
		c.Redirect(http.StatusFound, image.URL)
	}
}
