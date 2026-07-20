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
// 制作库：作者编辑文章的数据库，存储草稿和已发表文章的完整内容
type ArticleAuthorRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error) // 在制作库创建文章
	Update(ctx context.Context, article domain.Article) error          // 在制作库更新文章
}

// articleAuthorRepository 制作库仓储实现类（V1版本预留，当前未接入生产路径）
// V1架构下，制作库和线上库是分离的两个Repository，由Service层手动协调双写
type articleAuthorRepository struct {
	dao       newDAO.ArticleDAO // 通用文章DAO（当前实际使用的）
	authorDAO newDAO.AuthorDAO  // 制作库专属DAO（预留，未实现双库逻辑）
	readerDAO newDAO.ReaderDAO  // 线上库专属DAO（预留，未实现双库逻辑）
}

// NewArticleAuthorRepository 创建制作库仓储实例（V1版本预留）
// 参数中的 authorDAO/readerDAO 是为V1双库架构预留的，当前实现未使用
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
// 注意：当前实现只写入制作库，线上库需要由 Service 层单独调用 ArticleReaderRepository.Save
func (ar *articleAuthorRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, newDAO.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
	return id, err
}

// Update 在制作库更新文章，将领域模型转换为DAO实体后更新数据库
// 注意：当前实现只更新制作库，线上库需要由 Service 层单独调用 ArticleReaderRepository.Update
func (ar *articleAuthorRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.UpdateById(ctx, newDAO.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
	})
}
