package handlers

import (
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/GunarsK-portfolio/files-api/internal/storage"
)

type Handler struct {
	repo    repository.Repository
	storage *storage.Storage
	cfg     *config.Config
}

func New(repo repository.Repository, storage *storage.Storage, cfg *config.Config) *Handler {
	return &Handler{
		repo:    repo,
		storage: storage,
		cfg:     cfg,
	}
}
