package article

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
)

type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
}

type articleRepository struct {
	dao dao.ArticleDAO
}

func NewArticleRepository(dao dao.ArticleDAO) ArticleRepository {
	return &articleRepository{
		dao: dao,
	}
}

func (ar *articleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

func (ar *articleRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.Update(ctx, dao.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
