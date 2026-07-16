package oss

import (
	"blog_api/src/model"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// AliyunOSSService 实现了 OSSService 接口，用于阿里云 OSS
type AliyunOSSService struct {
	client *oss.Client
	config *model.OSSConfig
}

// NewAliyunOSSService 创建一个新的 AliyunOSSService 实例
func NewAliyunOSSService(cfg *model.OSSConfig) (OSSService, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client: %w", err)
	}
	return &AliyunOSSService{
		client: client,
		config: cfg,
	}, nil
}

// Upload stores a stream in Aliyun OSS.
func (s *AliyunOSSService) Upload(ctx context.Context, input UploadInput) (string, string, error) {
	// Debug log to check configuration
	fmt.Printf("[AliyunOSS] Uploading file. Config CustomDomain: '%s', Bucket: '%s'\n", s.config.CustomDomain, s.config.Bucket)

	bucket, err := s.client.Bucket(s.config.Bucket)
	if err != nil {
		return "", "", fmt.Errorf("failed to get oss bucket: %w", err)
	}
	objectKey := generateFilePath(s.config.Prefix, input.Name)
	err = bucket.PutObject(objectKey, input.Body, oss.ContentType(input.ContentType), oss.WithContext(ctx))
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file to oss: %w", err)
	}
	if s.config.CustomDomain != "" {
		customDomain := strings.TrimRight(s.config.CustomDomain, "/")
		return fmt.Sprintf("%s/%s", customDomain, objectKey), objectKey, nil
	}

	// 否则，返回标准的 OSS 访问 URL
	encodedObjectKey := url.PathEscape(objectKey)
	encodedObjectKey = strings.ReplaceAll(encodedObjectKey, "%2F", "/")
	return fmt.Sprintf("https://%s.%s/%s", s.config.Bucket, s.config.Endpoint, encodedObjectKey), objectKey, nil
}

// DeleteFile 实现了从阿里云 OSS 删除文件的逻辑
func (s *AliyunOSSService) DeleteFile(objectKey string) error {
	cleanKey := strings.TrimLeft(objectKey, "/")
	cleanPrefix := strings.Trim(s.config.Prefix, "/")
	// 安全检查：确保只删除配置路径下的文件
	if cleanPrefix != "" && !strings.HasPrefix(cleanKey, cleanPrefix) {
		return fmt.Errorf("delete operation is not allowed for this object key: %s", objectKey)
	}
	bucket, err := s.client.Bucket(s.config.Bucket)
	if err != nil {
		return fmt.Errorf("failed to get oss bucket: %w", err)
	}

	err = bucket.DeleteObject(cleanKey)
	if err != nil {
		return fmt.Errorf("failed to delete object from oss: %w", err)
	}
	return nil
}
