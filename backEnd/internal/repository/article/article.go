package article

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	newDAO "GoBook/internal/repository/dao/article"
	"GoBook/pkg/logger"
	"context"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"gorm.io/gorm"
)

// ArticleRepository 文章仓储接口，定义文章数据访问操作
type ArticleRepository interface {
	Create(ctx context.Context, article domain.Article) (int64, error)     // 创建文章
	Update(ctx context.Context, article domain.Article) error              // 更新文章
	Sync(ctx context.Context, article domain.Article) (int64, error)       // 同步文章到制作库和线上库
	SyncStatus(ctx context.Context, article domain.Article) (int64, error) // 同步更新两库的文章状态
	List(ctx context.Context, uid int64, offset int, limit int) ([]domain.Article, error)
	GetById(ctx context.Context, id int64) (domain.Article, error)
	GetByPubId(ctx context.Context, id int64) (domain.Article, error)
}

// articleRepository 文章仓储实现类
type articleRepository struct {
	dao          newDAO.ArticleDAO // 文章DAO，操作单一库
	authorDAO    newDAO.AuthorDAO  // 制作库DAO（预留V1版本）
	readerDAO    newDAO.ReaderDAO  // 线上库DAO（预留V1版本）
	db           *gorm.DB          // 数据库连接（预留V2事务版本）
	articleCache cache.ArticleCache
	l            logger.LoggerV1
}

// NewArticleRepository 创建文章仓储实例
func NewArticleRepository(
	dao newDAO.ArticleDAO,
	authorDAO newDAO.AuthorDAO,
	readerDAO newDAO.ReaderDAO,
	articleCache cache.ArticleCache,
	l logger.LoggerV1,
) ArticleRepository {
	return &articleRepository{
		dao:          dao,
		authorDAO:    authorDAO,
		readerDAO:    readerDAO,
		articleCache: articleCache,
		l:            l,
	}
}

// Create 创建文章，将领域模型转换为DAO实体后插入数据库
func (ar *articleRepository) Create(ctx context.Context, article domain.Article) (int64, error) {
	defer func() {
		//清空缓存
		ar.articleCache.DelFirstPage(ctx, article.Author.Id)
	}()
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
	defer func() {
		//清空缓存
		ar.articleCache.DelFirstPage(ctx, article.Author.Id)
	}()
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
	defer func() {
		//清空缓存
		ar.articleCache.DelFirstPage(ctx, article.Author.Id)
	}()
	return ar.dao.Sync(ctx, ar.toEntity(article))
}

// SyncStatus 同步更新两库的文章状态，事务内完成（委托给DAO层处理）
func (ar *articleRepository) SyncStatus(ctx context.Context, article domain.Article) (int64, error) {
	defer func() {
		//清空缓存
		ar.articleCache.DelFirstPage(ctx, article.Author.Id)
	}()
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

func (ar *articleRepository) List(ctx context.Context, authorId int64, offset int, limit int) ([]domain.Article, error) {
	//缓存设计
	//1.缓存第一页
	if offset == 0 && limit <= 100 {
		data, err := ar.articleCache.GetFirstPage(ctx, authorId)
		if err == nil {
			return data, nil
		}
	}

	res, err := ar.dao.GetByAuthor(ctx, authorId, offset, limit)
	if err != nil {
		return nil, err
	}
	data := slice.Map[newDAO.Article, domain.Article](
		res,
		func(idx int, src newDAO.Article) domain.Article {
			return ar.toDomain(src)
		},
	)

	//回写缓存，可以同步也可以异步
	go func() {
		err = ar.articleCache.SetFirstPage(ctx, authorId, data)
		ar.l.Error("回写缓存失败！", logger.Error(err))
	}()

	return data, nil
}

func (ar *articleRepository) GetById(ctx context.Context, id int64) (domain.Article, error) {
	res, err := ar.dao.GetById(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	data := ar.toDomain(res)
	return data, nil
}

func (ar *articleRepository) GetByPubId(ctx context.Context, id int64) (domain.Article, error) {
	res, err := ar.dao.GetByPubId(ctx, id)
	if err != nil {
		return domain.Article{}, err
	}
	data := ar.toDomain(newDAO.Article(res))
	return data, nil
}

// toDomain 将DAO实体转换为领域模型
func (ar *articleRepository) toDomain(article newDAO.Article) domain.Article {
	return domain.Article{
		Id:      article.Id,
		Title:   article.Title,
		Content: article.Content,
		Status:  domain.ArticleStatus(article.Status),
		Author: domain.Author{
			Id: article.AuthorId,
		},
		Ctime: time.UnixMilli(article.Ctime),
		Utime: time.UnixMilli(article.Utime),
	}
}

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
