package repository

import (
	"time"

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

type StorageFile struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	S3Key     string    `json:"s3Key" gorm:"column:s3_key"`
	S3Bucket  string    `json:"s3Bucket" gorm:"column:s3_bucket"`
	FileName  string    `json:"fileName" gorm:"column:file_name"`
	FileSize  int64     `json:"fileSize" gorm:"column:file_size"`
	MimeType  string    `json:"mimeType" gorm:"column:mime_type"`
	FileType  string    `json:"fileType" gorm:"column:file_type"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
}

func (StorageFile) TableName() string {
	return "storage.files"
}

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
