package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
)

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
}

type articleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(repo repository.ArticleRepository) ArticleService {
	return &articleService{
		repo: repo,
	}
}

func (as *articleService) Save(ctx context.Context, article domain.Article) (int64, error) {
	if article.Id > 0 {
		err := as.repo.Update(ctx, article)
		return article.Id, err
	}
	id, err := as.repo.Create(ctx, article)
	return id, err
}
