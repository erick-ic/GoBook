package service

import (
	"GoBook/internal/domain"
	events "GoBook/internal/events/article"
	"GoBook/internal/repository/article"
	"GoBook/pkg/logger"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// ArticleService 文章服务接口，定义文章的核心业务操作
// 调用链路：HTTP Handler → ArticleService → ArticleRepository → ArticleDAO

type ArticleService interface {
	Save(ctx context.Context, article domain.Article) (int64, error)                                             // 保存文章草稿（强制未发表状态）
	Publish(ctx context.Context, article domain.Article) (int64, error)                                          // 发表文章（同步到制作库和线上库）
	Withdraw(ctx context.Context, articleId, Uid int64) (int64, error)                                           // 撤回文章（同步更新两库状态）
	List(ctx context.Context, uid int64, offset int, limit int) ([]domain.Article, error)                        // 按作者分页查询文章列表
	GetById(ctx context.Context, id int64) (domain.Article, error)                                               // 查询文章详情（制作库）
	GetByPubId(ctx *gin.Context, articleId, Uid int64) (domain.Article, error)                                   // 查询已发表文章（线上库）+ 发送阅读事件
	ListPublishedArticles(ctx context.Context, start time.Time, offset int, limit int) ([]domain.Article, error) // 查询已发表文章（排行榜用）
}

// articleService 文章服务实现类
type articleService struct {
	repo     article.ArticleRepository // 文章仓储接口，负责数据库操作和缓存管理
	producer events.Producer           // Kafka 生产者，用于发送阅读事件
}

// ListPublishedArticles 查询已发表文章（排行榜用）
// 参数 start 指定起始时间，只返回该时间之后发布的文章
func (as *articleService) ListPublishedArticles(ctx context.Context, start time.Time, offset int, limit int) ([]domain.Article, error) {
	return as.repo.ListPublishedArticles(ctx, start, offset, limit)
}

// NewArticleService 创建文章服务实例
func NewArticleService(repo article.ArticleRepository, producer events.Producer) ArticleService {
	return &articleService{
		repo:     repo,
		producer: producer,
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
// 前端只需传文章 id（新建时 id=0），DAO 层会自行处理完整数据的获取
func (as *articleService) Publish(ctx context.Context, article domain.Article) (int64, error) {
	article.Status = domain.ArticleStatusPublished
	return as.repo.Sync(ctx, article)
}

// Withdraw 撤回文章，将状态改为未发表，同步更新两库状态
func (as *articleService) Withdraw(ctx context.Context, articleId, Uid int64) (int64, error) {
	status := domain.ArticleStatusUnPublished
	return as.repo.SyncStatus(ctx, articleId, Uid, status)
}

func (as *articleService) List(ctx context.Context, authorId int64, offset int, limit int) ([]domain.Article, error) {
	// 按作者分页查询文章列表，Repository 层负责缓存策略（第一页缓存）
	return as.repo.List(ctx, authorId, offset, limit)
}

func (as *articleService) GetById(ctx context.Context, id int64) (domain.Article, error) {
	// 查询文章详情（从制作库获取），用于编辑页面展示
	return as.repo.GetById(ctx, id)
}

// GetByPubId 获取已发表的文章详情，同时异步发送阅读事件到 Kafka
// 调用链路：用户访问 /pub/:id → PubDetail Handler → GetByPubId
//
// 阅读计数解耦方案：
//   - 不再在 Handler 层直接调用 IncrReadCnt（避免阻塞响应）
//   - 而是通过 Kafka 异步解耦：Service 层发事件 → 消费者异步更新阅读数
//   - 优点：响应快，不阻塞用户；缺点：阅读数有短暂延迟（最终一致）
//
// 注意：ctx 使用 *gin.Context 而非 context.Context，因为 producer.ProduceReadEvent 需要
func (as *articleService) GetByPubId(ctx *gin.Context, articleId, Uid int64) (domain.Article, error) {
	// 先查询文章详情（从线上库获取）
	art, err := as.repo.GetByPubId(ctx, articleId)
	if err == nil {
		// 查询成功后，异步发送阅读事件到 Kafka
		// 使用 goroutine 非阻塞发送，避免影响用户阅读体验
		go func() {
			er := as.producer.ProduceReadEvent(
				events.ReadEvent{
					ArticleId: articleId,
					Uid:       Uid,
				})
			if er != nil {
				// TODO: 写入日志（当前静默失败，生产环境应记录错误）
			}
		}()
	}
	return art, err
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
