package repository

import (
	"context"

	commonModels "github.com/GunarsK-portfolio/portfolio-common/models"
	"gorm.io/gorm"
)

type Repository interface {
	CreateFile(ctx context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*StorageFile, error)
	GetFileByID(ctx context.Context, id int64) (*StorageFile, error)
	GetFileByKey(ctx context.Context, bucket, key string) (*StorageFile, error)
	DeleteFile(ctx context.Context, id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

type StorageFile = commonModels.StorageFile

func (r *repository) CreateFile(ctx context.Context, bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*StorageFile, error) {
	file := &StorageFile{
		S3Key:    key,
		S3Bucket: bucket,
		FileName: fileName,
		FileSize: fileSize,
		MimeType: mimeType,
		FileType: fileType,
	}

	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return nil, err
	}

	return file, nil
}

func (r *repository) GetFileByID(ctx context.Context, id int64) (*StorageFile, error) {
	var file StorageFile
	if err := r.db.WithContext(ctx).First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *repository) GetFileByKey(ctx context.Context, bucket, key string) (*StorageFile, error) {
	var file StorageFile
	if err := r.db.WithContext(ctx).Where("s3_bucket = ? AND s3_key = ?", bucket, key).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *repository) DeleteFile(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&StorageFile{}, id).Error
}
