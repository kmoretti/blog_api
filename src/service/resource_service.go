package service

import (
	"blog_api/src/model"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// ResourceService 提供了处理资源（如文件上传）的服务。
type ResourceService struct {
	config *model.Config
}

var (
	ErrInvalidResourcePath = errors.New("invalid resource path")
	ErrProtectedResource   = errors.New("protected resource")
	ErrResourceNotFound    = errors.New("resource not found")
)

// NewResourceService 创建一个新的 ResourceService 实例。
func NewResourceService(cfg *model.Config) *ResourceService {
	return &ResourceService{config: cfg}
}

// SaveFile 保存上传的文件。
// 它会检查文件扩展名是否在白名单内，并清理目标路径以防止路径遍历。
// overwrite 参数决定如果文件已存在，是覆盖它还是生成一个新名字。
func (s *ResourceService) SaveFile(file *multipart.FileHeader, subPath string, overwrite bool) (string, string, error) {
	src, err := file.Open()
	if err != nil {
		return "", "", fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()
	return s.SaveReader(file.Filename, src, subPath, overwrite)
}

// SaveReader streams a resource into local storage.
func (s *ResourceService) SaveReader(filename string, src io.Reader, subPath string, overwrite bool) (string, string, error) {
	filePath, urlPath, err := s.prepareSavePath(filename, subPath, overwrite)
	if err != nil {
		return "", "", err
	}

	dst, err := os.CreateTemp(filepath.Dir(filePath), ".blog-api-upload-*")
	if err != nil {
		return "", "", fmt.Errorf("创建目标文件失败: %w", err)
	}
	tempPath := dst.Name()
	committed := false
	defer func() {
		if !committed {
			dst.Close()
			os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("保存文件失败: %w", err)
	}
	if err := dst.Close(); err != nil {
		return "", "", fmt.Errorf("关闭目标文件失败: %w", err)
	}
	if err := os.Chmod(tempPath, 0644); err != nil {
		return "", "", fmt.Errorf("设置目标文件权限失败: %w", err)
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		return "", "", fmt.Errorf("提交目标文件失败: %w", err)
	}
	committed = true

	return filePath, urlPath, nil
}

func (s *ResourceService) prepareSavePath(filename, subPath string, overwrite bool) (string, string, error) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if !s.isExtensionAllowed(ext) {
		return "", "", fmt.Errorf("文件类型 '%s' 不被允许", ext)
	}

	// 清理并构建保存路径，防止路径遍历攻击
	cleanSubPath := filepath.Clean(subPath)
	if strings.HasPrefix(cleanSubPath, "..") {
		return "", "", fmt.Errorf("无效的路径")
	}

	basePath := s.config.Data.Resource.Path
	if basePath == "" {
		basePath = "data/" // 默认路径
	}
	saveDir := filepath.Join(basePath, cleanSubPath)

	// 创建目录（如果不存在）
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 根据 overwrite 标志决定文件名
	var finalFilename string
	if overwrite {
		finalFilename = filename
	} else {
		finalFilename = s.findUniqueFilename(saveDir, filename)
	}
	filePath := filepath.Join(saveDir, finalFilename)

	// 从文件路径生成 URL
	urlPath := strings.TrimPrefix(filePath, strings.TrimSuffix(basePath, "/"))
	urlPath = filepath.ToSlash(urlPath)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	return filePath, urlPath, nil
}

// isExtensionAllowed 检查文件扩展名是否在白名单中。
func (s *ResourceService) isExtensionAllowed(ext string) bool {
	for _, allowedExt := range s.config.Safe.AllowExtension {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// findUniqueFilename 检查文件名是否重复，如果重复则添加后缀 (1), (2)...
func (s *ResourceService) findUniqueFilename(dir, filename string) string {
	filePath := filepath.Join(dir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return filename // 文件名不重复，直接返回
	}

	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	counter := 1
	for {
		// 生成新的文件名，例如: "image(1).png"
		newFilename := fmt.Sprintf("%s(%d)%s", baseName, counter, ext)
		newFilePath := filepath.Join(dir, newFilename)
		if _, err := os.Stat(newFilePath); os.IsNotExist(err) {
			return newFilename // 找到一个不重复的文件名
		}
		counter++
	}
}

// DeleteFile 删除指定路径的文件。
// 在删除前会进行严格的安全检查，以防止删除受保护的文件。
func (s *ResourceService) DeleteFile(filePath string) error {
	// 1. 解析资源根目录（通常是 data），删除目标必须位于该目录内
	basePath := s.config.Data.Resource.Path
	if basePath == "" {
		basePath = "data/"
	}
	absBasePath, err := filepath.Abs(filepath.Clean(basePath))
	if err != nil {
		return fmt.Errorf("获取资源根目录失败: %w", err)
	}

	relativePath := filepath.Clean(filePath)
	if relativePath == "." || relativePath == string(filepath.Separator) {
		return fmt.Errorf("%w: %s", ErrInvalidResourcePath, filePath)
	}
	// 将绝对输入路径按相对路径处理，避免绕过 basePath 校验
	relativePath = strings.TrimLeft(relativePath, `/\`)

	cleanPath, err := filepath.Abs(filepath.Join(absBasePath, relativePath))
	if err != nil {
		return fmt.Errorf("获取目标绝对路径失败: %w", err)
	}
	baseWithSep := absBasePath + string(filepath.Separator)
	if cleanPath != absBasePath && !strings.HasPrefix(cleanPath, baseWithSep) {
		return fmt.Errorf("%w: %s", ErrInvalidResourcePath, filePath)
	}

	// 2. 检查路径是否在受保护的目录内
	for _, protectedPath := range s.config.Safe.ExcludePaths {
		isProtected := false
		candidates := []string{}

		absProtectedPath, err := filepath.Abs(filepath.Clean(protectedPath))
		if err == nil {
			candidates = append(candidates, absProtectedPath)
		}

		// 兼容以资源根为基准的写法（如 "/config"）
		relativeProtectedPath := strings.TrimLeft(filepath.Clean(protectedPath), `/\`)
		if relativeProtectedPath != "" && relativeProtectedPath != "." {
			absProtectedFromBase, err := filepath.Abs(filepath.Join(absBasePath, relativeProtectedPath))
			if err == nil {
				candidates = append(candidates, absProtectedFromBase)
			}
		}

		for _, candidate := range candidates {
			candidateWithSep := candidate + string(filepath.Separator)
			if cleanPath == candidate || strings.HasPrefix(cleanPath, candidateWithSep) {
				isProtected = true
				break
			}
		}

		if isProtected {
			return fmt.Errorf("%w: '%s'", ErrProtectedResource, filePath)
		}
	}

	// 3. 检查文件是否存在
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: '%s'", ErrResourceNotFound, filePath)
	}

	// 4. 执行删除
	if err := os.Remove(cleanPath); err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}

	return nil
}

// GetFileOrDir 检索文件或列出目录的内容。
// 如果路径指向文件，它将返回文件路径和 nil 切片。
// 如果路径指向目录，它将返回一个包含目录内容的 FileInfo 切片。
// 如果路径不存在，则返回错误。
func (s *ResourceService) GetFileOrDir(relativePath string) (string, []model.FileInfo, error) {
	// 清理路径以防止路径遍历
	cleanPath := filepath.Clean(relativePath)
	if strings.HasPrefix(cleanPath, "..") {
		return "", nil, fmt.Errorf("无效的路径")
	}

	basePath := s.config.Data.Resource.Path
	if basePath == "" {
		basePath = "data/" // 默认路径
	}
	fullPath := filepath.Join(basePath, cleanPath)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, fmt.Errorf("资源未找到: %s", relativePath)
		}
		return "", nil, fmt.Errorf("访问资源时出错: %w", err)
	}

	if info.IsDir() {
		// 列出目录内容
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return "", nil, fmt.Errorf("读取目录失败: %w", err)
		}

		var fileInfos []model.FileInfo
		for _, entry := range entries {
			entryInfo, err := entry.Info()
			if err != nil {
				// 可以选择记录错误并继续
				continue
			}
			fileInfos = append(fileInfos, model.FileInfo{
				Name:    entry.Name(),
				Path:    filepath.ToSlash(filepath.Join(cleanPath, entry.Name())),
				IsDir:   entry.IsDir(),
				Size:    entryInfo.Size(),
				ModTime: entryInfo.ModTime(),
			})
		}
		return "", fileInfos, nil
	} else {
		// 返回文件路径
		return fullPath, nil, nil
	}
}
