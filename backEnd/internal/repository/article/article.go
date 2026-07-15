package article

import (
	"GoBook/internal/domain"
	newDAO "GoBook/internal/repository/dao/article"
	"context"

	"gorm.io/gorm"
)

// ArticleRepository 文章仓储接口，定义文章数据访问操作
type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)     // 创建文章
	Update(ctx context.Context, article domain.Article) error              // 更新文章
	Sync(ctx context.Context, article domain.Article) (int64, error)       // 同步文章到制作库和线上库
	SyncStatus(ctx context.Context, article domain.Article) (int64, error) // 同步更新两库的文章状态
}

// articleRepository 文章仓储实现类
type articleRepository struct {
	dao       newDAO.ArticleDAO // 文章DAO，操作单一库
	authorDAO newDAO.AuthorDAO  // 制作库DAO（预留V1版本）
	readerDAO newDAO.ReaderDAO  // 线上库DAO（预留V1版本）
	db        *gorm.DB          // 数据库连接（预留V2事务版本）
}

// NewArticleRepository 创建文章仓储实例
func NewArticleRepository(
	dao newDAO.ArticleDAO,
	authorDAO newDAO.AuthorDAO,
	readerDAO newDAO.ReaderDAO,
) ArticleRepository {
	return &articleRepository{
		dao:       dao,
		authorDAO: authorDAO,
		readerDAO: readerDAO,
	}
}

// Create 创建文章，将领域模型转换为DAO实体后插入数据库
func (ar *articleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	id, err := ar.dao.Insert(ctx, newDAO.Article{
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
		Status:   article.Status.ToUint8(),
	})
	return id, err
}

// Update 更新文章，将领域模型转换为DAO实体后更新数据库
func (ar *articleRepository) Update(ctx context.Context, article domain.Article) error {
	return ar.dao.UpdateById(ctx, newDAO.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
		Status:   article.Status.ToUint8(),
	})
}

// Sync 同步文章到制作库和线上库，事务内完成（委托给DAO层处理）
func (ar *articleRepository) Sync(ctx context.Context, article domain.Article) (int64, error) {
	return ar.dao.Sync(ctx, ar.toEntity(article))
}

// SyncStatus 同步更新两库的文章状态，事务内完成（委托给DAO层处理）
func (ar *articleRepository) SyncStatus(ctx context.Context, article domain.Article) (int64, error) {
	return ar.dao.SyncStatus(ctx, ar.toEntity(article))
}

// SyncV1 预留：V1版本同步方案，手动操作两个DAO（无事务）
// 先写入制作库，再写入线上库，适用于异构存储场景
//func (ar *articleRepository) SyncV1(ctx context.Context, article domain.Article) (int64, error) {
//	var (
//		id  = article.Id
//		err error
//	)
//	artn := ar.toEntity(article)
//	if id > 0 {
//		err = ar.authorDAO.UpdateById(ctx, artn)
//	} else {
//		id, err = ar.authorDAO.Insert(ctx, artn)
//	}
//	if err != nil {
//		return id, err
//	}
//	err = ar.readerDAO.Upsert(ctx, artn)
//	return id, err
//}

// SyncV2 预留：V2版本同步方案，手动开启事务
// 使用defer Rollback确保事务不会悬而未决
//func (ar *articleRepository) SyncV2(ctx context.Context, article domain.Article) (int64, error) {
//	tx := ar.db.WithContext(ctx).Begin()
//	if tx.Error != nil {
//		return 0, tx.Error
//	}
//	defer tx.Rollback()
//
//	author := newDAO.NewAuthorDAO(tx)
//	reader := newDAO.NewReaderDAO(tx)
//
//	var (
//		id  = article.Id
//		err error
//	)
//	artn := ar.toEntity(article)
//	if id > 0 {
//		err = author.UpdateById(ctx, artn)
//	} else {
//		id, err = author.Insert(ctx, artn)
//	}
//	if err != nil {
//		return id, err
//	}
//	err = reader.Upsert(ctx, artn)
//	tx.Commit()
//	return id, err
//}

// toEntity 将领域模型转换为DAO实体
func (ar *articleRepository) toEntity(article domain.Article) newDAO.Article {
	return newDAO.Article{
		Id:       article.Id,
		Title:    article.Title,
		Content:  article.Content,
		AuthorId: article.Author.Id,
		Status:   article.Status.ToUint8(),
	}
}
