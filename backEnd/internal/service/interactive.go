package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
)

type InteractiveService interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	Like(ctx context.Context, biz string, articleId int64, uid int64) error
	CancelLike(ctx context.Context, biz string, articleId int64, uid int64) error
	Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
}

func (is *interactiveService) Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error) {
	inter, err := is.repo.Get(ctx, biz, id)
	if err != nil {
		return domain.Interactive{}, err
	}
	return inter, nil
}

func (is *interactiveService) Like(ctx context.Context, biz string, articleId int64, uid int64) error {
	return is.repo.IncrLike(ctx, biz, articleId, uid)
}

func (is *interactiveService) CancelLike(ctx context.Context, biz string, articleId int64, uid int64) error {
	return is.repo.DecrLike(ctx, biz, articleId, uid)
}

func (is *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return is.repo.IncrReadCnt(ctx, biz, bizId)
}

func NewInteractiveService(repo repository.InteractiveRepository) InteractiveService {
	return &interactiveService{
		repo: repo,
	}
}
