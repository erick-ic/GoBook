// ============================================================
// 预留代码（V1版本）：制作库仓储
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

// ArticleAuthorRepository 制作库仓储接口，定义制作库的文章数据访问操作（V1版本预留）
type ArticleAuthorRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error) // 创建文章
	Update(ctx context.Context, article domain.Article) error          // 更新文章
}

// articleAuthorRepository 制作库仓储实现类（V1版本预留，当前未接入生产路径）
type articleAuthorRepository struct {
	dao       newDAO.ArticleDAO // 文章DAO
	authorDAO newDAO.AuthorDAO  // 制作库DAO
	readerDAO newDAO.ReaderDAO  // 线上库DAO
}

// NewArticleAuthorRepository 创建制作库仓储实例（V1版本预留）
func NewArticleAuthorRepository(
	dao newDAO.ArticleDAO,
	authorDAO newDAO.AuthorDAO,
	readerDAO newDAO.ReaderDAO,
) ArticleAuthorRepository {
	return &articleAuthorRepository{
		dao:       dao,
		authorDAO: authorDAO,
		readerDAO: readerDAO,
	}
}

// Create 在制作库创建文章，将领域模型转换为DAO实体后插入数据库
func (ar *articleAuthorRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, newDAO.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

// Update 在制作库更新文章，将领域模型转换为DAO实体后更新数据库
func (ar *articleAuthorRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.UpdateById(ctx, newDAO.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
