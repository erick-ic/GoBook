// ============================================================
// 预留代码（V1版本）：线上库仓储
// 当前生产路径走 articleRepository + articleDAO（闭包事务方案），
// 本文件用于演示"双 Repository 非事务双写"的 V1 架构，
// 接口被 articleServiceV1 和 mock 测试依赖，实现未接入 Wire。
// ============================================================

package article

import (
	"GoBook/internal/domain"
	newDAO "GoBook/internal/repository/dao/article"
	"context"
)

// ArticleReaderRepository 线上库仓储接口，定义线上库的文章数据访问操作（V1版本预留）
type ArticleReaderRepository interface {
	//Create(ctx context.Context, article domain.Article) (int64, error)
	Save(ctx context.Context, article domain.Article) (int64, error) // 保存文章（插入或更新）
	Update(ctx context.Context, article domain.Article) error        // 更新文章
}

// articleReaderRepository 线上库仓储实现类（V1版本预留，当前未接入生产路径）
type articleReaderRepository struct {
	dao newDAO.ArticleDAO // 文章DAO
}

// NewArticleReaderRepository 创建线上库仓储实例（V1版本预留）
func NewArticleReaderRepository(dao newDAO.ArticleDAO) ArticleReaderRepository {
	return &articleReaderRepository{
		dao: dao,
	}
}

// Save 保存文章到线上库，将领域模型转换为DAO实体后插入数据库
func (ar *articleReaderRepository) Save(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, newDAO.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

// Update 更新线上库文章，将领域模型转换为DAO实体后更新数据库
func (ar *articleReaderRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.UpdateById(ctx, newDAO.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
