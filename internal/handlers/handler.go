package handlers

import (
	"github.com/GunarsK-portfolio/files-api/internal/config"
	"github.com/GunarsK-portfolio/files-api/internal/repository"
	"github.com/GunarsK-portfolio/files-api/internal/storage"
	commonrepo "github.com/GunarsK-portfolio/portfolio-common/repository"
)

type Handler struct {
	repo          repository.Repository
	storage       storage.ObjectStore
	cfg           *config.Config
	actionLogRepo commonrepo.ActionLogRepository
}

func New(repo repository.Repository, storage storage.ObjectStore, cfg *config.Config, actionLogRepo commonrepo.ActionLogRepository) *Handler {
	return &Handler{
		repo:          repo,
		storage:       storage,
		cfg:           cfg,
		actionLogRepo: actionLogRepo,
	}
}
