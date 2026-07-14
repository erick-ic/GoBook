package article

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
)

type ArticleReaderRepository interface {
	//Create(ctx context.Context, article domain.Article) (int64, error)
	Save(ctx context.Context, article domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) error
}

type articleReaderRepository struct {
	dao dao.ArticleDAO
}

func NewArticleReaderRepository(dao dao.ArticleDAO) ArticleReaderRepository {
	return &articleReaderRepository{
		dao: dao,
	}
}

func (ar *articleReaderRepository) Save(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, dao.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

func (ar *articleReaderRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.Update(ctx, dao.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
