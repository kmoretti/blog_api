package oss

import (
	"blog_api/src/config"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadInput describes a replayable object upload.
type UploadInput struct {
	Name        string
	ContentType string
	Size        int64
	Body        io.ReadSeeker
}

// OSSService defines object storage operations.
type OSSService interface {
	// Upload stores input and returns its public URL and object key.
	Upload(ctx context.Context, input UploadInput) (string, string, error)
	// DeleteFile 从 OSS 删除文件
	// objectKey: 文件在 OSS 中的 key
	DeleteFile(objectKey string) error
}

// ValidateOSSConfig checks whether the OSS config is usable by making a minimal request.
func ValidateOSSConfig() error {
	cfg := config.GetConfig()
	if !cfg.OSS.Enable {
		return nil
	}

	switch cfg.OSS.Provider {
	case "aliyun":
		client, err := oss.New(cfg.OSS.Endpoint, cfg.OSS.AccessKeyID, cfg.OSS.AccessKeySecret)
		if err != nil {
			return fmt.Errorf("failed to create aliyun oss client: %w", err)
		}
		bucket, err := client.Bucket(cfg.OSS.Bucket)
		if err != nil {
			return fmt.Errorf("failed to get oss bucket: %w", err)
		}
		if _, err := bucket.ListObjects(oss.MaxKeys(1)); err != nil {
			return fmt.Errorf("failed to list oss objects: %w", err)
		}
		return nil
	case "s3":
		s3Client, err := newS3Client(&cfg.OSS)
		if err != nil {
			return err
		}
		_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
			Bucket: &cfg.OSS.Bucket,
		})
		if err != nil {
			return fmt.Errorf("failed to head s3 bucket: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported OSS provider: %s", cfg.OSS.Provider)
	}
}

// NewOSSService 是一个工厂函数，根据配置创建并返回一个具体的 OSSService 实例
func NewOSSService() (OSSService, error) {
	cfg := config.GetConfig()
	if !cfg.OSS.Enable {
		return nil, fmt.Errorf("OSS service is not enabled in the configuration")
	}

	switch cfg.OSS.Provider {
	case "aliyun":
		return NewAliyunOSSService(&cfg.OSS)
	// case "tencent":
	// 	// 可以在此添加腾讯云 COS 的实现
	// 	// return nil, fmt.Errorf("tencent COS provider is not yet implemented")
	case "s3":
		return NewS3OSSService(&cfg.OSS)
	default:
		return nil, fmt.Errorf("unsupported OSS provider: %s", cfg.OSS.Provider)
	}
}

// generateFilePath 生成在 OSS 中存储的文件路径
// 使用 prefix 和原始文件名，并可以添加时间戳或 UUID 以避免冲突
func generateFilePath(prefix, originalFilename string) string {
	// 确保路径分隔符为 '/', 以兼容 OSS
	ossFriendlyFilename := strings.ReplaceAll(originalFilename, "\\", "/")
	timestamp := time.Now().Unix()
	dir, file := path.Split(ossFriendlyFilename)
	uniqueFile := fmt.Sprintf("%d-%s", timestamp, file)
	fullPath := strings.Trim(path.Join(dir, uniqueFile), "/")

	cleanPrefix := strings.Trim(prefix, "/")
	if cleanPrefix == "" {
		return fullPath
	}

	return fmt.Sprintf("%s/%s", cleanPrefix, fullPath)
}
