package repository

import (
	commonModels "github.com/GunarsK-portfolio/portfolio-common/models"
	"gorm.io/gorm"
)

type Repository interface {
	CreateFile(bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*StorageFile, error)
	GetFileByID(id int64) (*StorageFile, error)
	GetFileByKey(bucket, key string) (*StorageFile, error)
	DeleteFile(id int64) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

type StorageFile = commonModels.StorageFile

func (r *repository) CreateFile(bucket, key, fileName, fileType string, fileSize int64, mimeType string) (*StorageFile, error) {
	file := &StorageFile{
		S3Key:    key,
		S3Bucket: bucket,
		FileName: fileName,
		FileSize: fileSize,
		MimeType: mimeType,
		FileType: fileType,
	}

	if err := r.db.Create(file).Error; err != nil {
		return nil, err
	}

	return file, nil
}

func (r *repository) GetFileByID(id int64) (*StorageFile, error) {
	var file StorageFile
	if err := r.db.First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *repository) GetFileByKey(bucket, key string) (*StorageFile, error) {
	var file StorageFile
	if err := r.db.Where("s3_bucket = ? AND s3_key = ?", bucket, key).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *repository) DeleteFile(id int64) error {
	return r.db.Delete(&StorageFile{}, id).Error
}
