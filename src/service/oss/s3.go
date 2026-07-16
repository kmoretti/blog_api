package oss

import (
	"blog_api/src/model"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3OSSService 实现了 OSSService 接口，用于与 S3 兼容的对象存储
type S3OSSService struct {
	client   *s3.Client
	uploader *manager.Uploader
	config   *model.OSSConfig
}

// NewS3OSSService 创建一个新的 S3OSSService 实例
func NewS3OSSService(cfg *model.OSSConfig) (OSSService, error) {
	s3Client, err := newS3Client(cfg)
	if err != nil {
		return nil, err
	}
	uploader := manager.NewUploader(s3Client)

	return &S3OSSService{
		client:   s3Client,
		uploader: uploader,
		config:   cfg,
	}, nil
}

func newS3Client(cfg *model.OSSConfig) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load s3 config: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	}), nil
}

// Upload stores a stream in S3.
func (s *S3OSSService) Upload(ctx context.Context, input UploadInput) (string, string, error) {
	// 生成在 OSS 中的存储路径
	objectKey := generateFilePath(s.config.Prefix, input.Name)
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(objectKey),
		Body:          input.Body,
		ContentType:   aws.String(input.ContentType),
		ContentLength: aws.Int64(input.Size),
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload file to s3: %w", err)
	}

	// 根据配置返回访问 URL
	if s.config.CustomDomain != "" {
		return fmt.Sprintf("%s/%s", s.config.CustomDomain, objectKey), objectKey, nil
	}

	// 否则，返回标准的 S3 访问 URL
	encodedObjectKey := url.PathEscape(objectKey)
	if s.config.Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.config.Endpoint, s.config.Bucket, encodedObjectKey), objectKey, nil
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.config.Bucket, s.config.Region, encodedObjectKey), objectKey, nil
}

// DeleteFile 实现了从 S3 删除文件的逻辑
func (s *S3OSSService) DeleteFile(objectKey string) error {
	cleanKey := strings.TrimLeft(objectKey, "/")
	cleanPrefix := strings.Trim(s.config.Prefix, "/")
	// 安全检查：确保只删除配置路径下的文件
	if cleanPrefix != "" && !strings.HasPrefix(cleanKey, cleanPrefix) {
		return fmt.Errorf("delete operation is not allowed for this object key: %s", objectKey)
	}

	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(cleanKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object from s3: %w", err)
	}

	return nil
}
