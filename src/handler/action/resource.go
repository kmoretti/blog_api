package handlerAction

import (
	"blog_api/src/model"
	"blog_api/src/service"
	"blog_api/src/service/oss"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxUploadRequestBytes = int64(65 << 20)

func limitUploadBody(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadRequestBytes)
}

func uploadErrorStatus(err error) int {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

// ResourceHandler 封装了处理资源相关请求的逻辑。
type ResourceHandler struct {
	resourceService *service.ResourceService
	ossService      oss.OSSService
}

// NewResourceHandler 创建一个新的 ResourceHandler 实例。
func NewResourceHandler(cfg *model.Config, ossService oss.OSSService) *ResourceHandler {
	return &ResourceHandler{
		resourceService: service.NewResourceService(cfg),
		ossService:      ossService,
	}
}

// UploadResourceLocal 处理本地文件上传请求。
func (h *ResourceHandler) UploadResourceLocal(c *gin.Context) {
	limitUploadBody(c)
	// 从表单中获取文件
	file, err := c.FormFile("file")
	if err != nil {
		status := uploadErrorStatus(err)
		c.JSON(status, model.NewErrorResponse(status, "获取文件失败: "+err.Error()))
		return
	}

	// 绑定表单字段
	var req model.UploadResourceReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的表单参数: "+err.Error()))
		return
	}

	localPath, urlPath, err := h.resourceService.SaveFile(file, req.Path, req.Overwrite)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{
		"url":        urlPath,
		"local_path": localPath,
	}))
}

// UploadResourceOSS 处理 OSS 文件上传请求。
func (h *ResourceHandler) UploadResourceOSS(c *gin.Context) {
	if h.ossService == nil {
		c.JSON(http.StatusNotImplemented, model.NewErrorResponse(http.StatusNotImplemented, "OSS 服务未启用"))
		return
	}
	limitUploadBody(c)

	var req model.UploadResourceReq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "无效的表单参数: "+err.Error()))
		return
	}

	// 从表单中获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		status := uploadErrorStatus(err)
		c.JSON(status, model.NewErrorResponse(status, "获取文件失败: "+err.Error()))
		return
	}
	defer file.Close()

	baseName := path.Base(header.Filename)
	if cleanPath := strings.Trim(req.Path, "/"); cleanPath != "" {
		header.Filename = path.Join(cleanPath, baseName)
	} else {
		header.Filename = baseName
	}

	url, objectKey, err := h.ossService.Upload(c.Request.Context(), oss.UploadInput{
		Name:        header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Size:        header.Size,
		Body:        file,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, "上传到 OSS 失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse(gin.H{
		"url":       url,
		"objectKey": objectKey,
	}))
}

// DeleteResourceLocal 处理本地文件删除请求。
func (h *ResourceHandler) DeleteResourceLocal(c *gin.Context) {
	// 从 URL 通配符参数中获取文件路径
	filePath := c.Param("file_path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "文件路径不能为空"))
		return
	}

	// Gin 的通配符参数会包含一个前导斜杠，需要去掉
	filePath = filePath[1:]
	if err := h.resourceService.DeleteFile(filePath); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidResourcePath):
			c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, err.Error()))
		case errors.Is(err, service.ErrProtectedResource):
			c.JSON(http.StatusForbidden, model.NewErrorResponse(http.StatusForbidden, err.Error()))
		case errors.Is(err, service.ErrResourceNotFound):
			c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, err.Error()))
		default:
			c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		}
		return
	}
	c.JSON(http.StatusOK, model.NewSuccessResponse("文件删除成功"))
}

// DeleteResourceOSS 处理 OSS 文件删除请求。
func (h *ResourceHandler) DeleteResourceOSS(c *gin.Context) {
	if h.ossService == nil {
		c.JSON(http.StatusNotImplemented, model.NewErrorResponse(http.StatusNotImplemented, "OSS 服务未启用"))
		return
	}
	// 从 URL 通配符参数中获取文件路径
	objectKey := c.Param("file_path")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, model.NewErrorResponse(http.StatusBadRequest, "文件路径不能为空"))
		return
	}
	objectKey = objectKey[1:]

	if err := h.ossService.DeleteFile(objectKey); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewErrorResponse(http.StatusInternalServerError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.NewSuccessResponse("文件删除成功"))
}

// GetResource 处理获取文件或目录列表的请求。
func (h *ResourceHandler) GetResource(c *gin.Context) {
	filePath := c.Param("file_path")
	if filePath == "" {
		filePath = "/"
	} else {
		filePath = filePath[1:]
	}

	fullPath, fileInfos, err := h.resourceService.GetFileOrDir(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, model.NewErrorResponse(http.StatusNotFound, err.Error()))
		return
	}

	if fileInfos != nil {
		c.JSON(http.StatusOK, model.NewSuccessResponse(fileInfos))
	} else {
		c.File(fullPath)
	}
}
