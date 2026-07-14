package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/article"
	"GoBook/pkg/logger"
	"context"
)

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)
	Publish(ctx context.Context, article domain.Article) (int64, error)
}

type articleService struct {
	repo article.ArticleRepository
}

func NewArticleService(repo article.ArticleRepository) ArticleService {
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

func (as *articleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	//id, err := as.repo.Create(ctx, article)
	panic("implement me")
}

type ArticleServiceV1 interface {
	PublishV1(ctx context.Context, article domain.Article) (int64, error)
}

type articleServiceV1 struct {
	author article.ArticleAuthorRepository
	reader article.ArticleReaderRepository
	l      logger.LoggerV1
}

func NewArticleServiceV1(
	author article.ArticleAuthorRepository,
	reader article.ArticleReaderRepository,
	l logger.LoggerV1) ArticleServiceV1 {
	return &articleServiceV1{
		author: author,
		reader: reader,
		l:      l,
	}
}

func (as *articleServiceV1) PublishV1(ctx context.Context, article domain.Article) (int64, error) {
	var (
		id  = article.Id
		err error
	)
	if article.Id > 0 {
		err = as.author.Update(ctx, article)
		if err != nil {
			return 0, err
		}
	} else {
		id, err = as.author.Create(ctx, article)
		if err != nil {
			return 0, err
		}
	}
	//保证制作库和线上库相等
	article.Id = id

	//重试：
	for i := 0; i < 3; i++ {
		id, err = as.reader.Save(ctx, article)
		if err == nil {
			break
		}
		as.l.Error("部分失败：保存到线上库失败",
			logger.Int64("article_id", article.Id),
			logger.Error(err))
	}
	if err != nil {
		as.l.Error("部分失败：保存到线上库重试失败",
			logger.Int64("article_id", article.Id),
			logger.Error(err))
	}
	return id, err
}
