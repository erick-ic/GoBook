package article

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
)

type ArticleAuthorRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
}

type articleAuthorRepository struct {
	dao dao.ArticleDAO
}

func NewArticleAuthorRepository(dao dao.ArticleDAO) ArticleAuthorRepository {
	return &articleAuthorRepository{
		dao: dao,
	}
}

func (ar *articleAuthorRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

func (ar *articleAuthorRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.Update(ctx, dao.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
