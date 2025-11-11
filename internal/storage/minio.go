package storage

import (
	"context"
	"io"
	"strings"

	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	client *minio.Client
}

//nolint:staticcheck // Embedded field name required for clarity
func New(cfg *config.Config) (*Storage, error) {
	// Strip protocol from endpoint (MinIO client expects just hostname:port)
	endpoint := strings.TrimPrefix(cfg.S3Config.Endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3Config.AccessKey, cfg.S3Config.SecretKey, ""),
		Secure: cfg.S3Config.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	return &Storage{
		client: client,
	}, nil
}

func (s *Storage) GetObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	return s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
}

func (s *Storage) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *Storage) DeleteObject(ctx context.Context, bucket, key string) error {
	return s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func (s *Storage) StatObject(ctx context.Context, bucket, key string) (minio.ObjectInfo, error) {
	return s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
}
