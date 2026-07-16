package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/article"
	"GoBook/pkg/logger"
	"context"

	"github.com/gin-gonic/gin"
)

// ArticleService 文章服务接口，定义文章的核心业务操作
type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)     // 保存文章草稿
	Publish(ctx context.Context, article domain.Article) (int64, error)  // 发表文章
	Withdraw(ctx context.Context, article domain.Article) (int64, error) // 撤回文章
	List(ctx context.Context, uid int64, offset int, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)
	GetByPubId(ctx *gin.Context, id int64) (domain.Article, error)
}

// articleService 文章服务实现类
type articleService struct {
	repo article.ArticleRepository // 文章仓储接口
}

// NewArticleService 创建文章服务实例
func NewArticleService(repo article.ArticleRepository) ArticleService {
	return &articleService{
		repo: repo,
	}
}

// Save 保存文章草稿，强制状态为未发表
func (as *articleService) Save(ctx context.Context, article domain.Article) (int64, error) {
	article.Status = domain.ArticleStatusUnPublished
	if article.Id > 0 {
		err := as.repo.Update(ctx, article)
		return article.Id, err
	}
	id, err := as.repo.Create(ctx, article)
	return id, err
}

// Publish 发表文章，强制状态为已发表，同步到制作库和线上库
func (as *articleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	article.Status = domain.ArticleStatusPublished
	return as.repo.Sync(ctx, article)
}

// ArticleServiceV1 文章服务V1版本接口，用于演示非事务双写方案
type ArticleServiceV1 interface {
	PublishV1(ctx context.Context, article domain.Article) (int64, error) // V1版本发表文章
}

// articleServiceV1 V1版本文章服务实现，采用非事务双写 + 重试策略
type articleServiceV1 struct {
	author article.ArticleAuthorRepository // 制作库仓储
	reader article.ArticleReaderRepository // 线上库仓储
	l      logger.LoggerV1                 // 日志记录器
}

// NewArticleServiceV1 创建V1版本文章服务实例
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

// PublishV1 V1版本发表文章，先写入制作库，再写入线上库（带重试）
// 若线上库写入失败会重试3次，记录错误日志但不回滚制作库
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
	article.Id = id

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

// Withdraw 撤回文章，将状态改为未发表，同步更新两库状态
func (as *articleService) Withdraw(ctx context.Context, article domain.Article) (int64, error) {
	article.Status = domain.ArticleStatusUnPublished
	return as.repo.SyncStatus(ctx, article)
}

func (as *articleService) List(ctx context.Context, authorId int64, offset int, limit int) ([]domain.Article, error) {
	return as.repo.List(ctx, authorId, offset, limit)
}

func (as *articleService) GetById(ctx context.Context, id int64) (domain.Article, error) {
	return as.repo.GetById(ctx, id)
}

func (as *articleService) GetByPubId(ctx *gin.Context, id int64) (domain.Article, error) {
	return as.repo.GetByPubId(ctx, id)
}
