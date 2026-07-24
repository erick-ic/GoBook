package article

import (
	"GoBook/internal/domain"
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ArticleDAO 文章数据访问对象接口，定义文章的数据库操作
type ArticleDAO interface {
	Insert(ctx context.Context, article Article) (int64, error)                                       // 插入文章记录
	UpdateById(ctx context.Context, article Article) error                                            // 根据ID更新文章
	Sync(ctx context.Context, article Article) (int64, error)                                         // 事务内同步文章到制作库和线上库
	Upsert(ctx context.Context, article PublishArticle) error                                         // 线上库插入或更新（UPSERT）
	SyncStatus(ctx context.Context, articleId, Uid int64, status domain.ArticleStatus) (int64, error) // 事务内同步更新两库的文章状态
	GetByAuthor(ctx context.Context, authorId int64, offset int, limit int) ([]Article, error)
	GetById(ctx context.Context, id int64) (Article, error)
	GetByPubId(ctx context.Context, id int64) (PublishArticle, error)
	ListPublishedArticles(ctx context.Context, start time.Time, offset int, limit int) ([]Article, error)
}

// articleDAO 文章数据访问对象实现类
type articleDAO struct {
	db *gorm.DB // 数据库连接
}

func (ad *articleDAO) ListPublishedArticles(ctx context.Context, start time.Time, offset int, limit int) ([]Article, error) {
	var articles []Article
	err := ad.db.WithContext(ctx).Where("utime < ?", start.UnixMilli()).
		Order("utime DESC").Offset(offset).Limit(limit).Find(&articles).Error
	return articles, err
}

// NewArticleDAO 创建文章数据访问对象实例
func NewArticleDAO(db *gorm.DB) ArticleDAO {
	return &articleDAO{
		db: db,
	}
}

// Insert 插入文章记录，自动设置创建时间和更新时间
func (ad *articleDAO) Insert(ctx context.Context, article Article) (int64, error) {
	now := time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := ad.db.WithContext(ctx).Create(&article).Error
	return article.Id, err
}

// UpdateById 根据ID更新文章，带上作者ID防止修改他人文章
// 更新字段包括标题、内容、状态和更新时间，若未命中记录则返回错误
//
// 注意：不能使用 db.Model(&article).Updates(map)，
// 因为 Model(&article) 会把 article 的零值字段（如空 title）作为 WHERE 条件，
// 导致匹配不到记录或清空已有数据。
// 正确做法：用 db.Model(&Article{}).Where(...).Updates(map) 显式指定条件和更新值。
func (ad *articleDAO) UpdateById(ctx context.Context, article Article) error {
	now := time.Now().UnixMilli()

	// 用空模型 + 显式 WHERE 条件，避免 GORM 把零值字段作为查询条件
	res := ad.db.WithContext(ctx).Model(&Article{}).
		Where("id = ? AND author_id = ?", article.Id, article.AuthorId).
		Updates(map[string]any{
			"title":   article.Title,
			"content": article.Content,
			"status":  article.Status,
			"utime":   now,
		})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("更新失败，可能是作者非法，id %d，author_id %d", article.Id, article.AuthorId)
	}
	return res.Error
}

// Sync 事务内同步文章到制作库和线上库，保证两库数据一致性
// 采用GORM闭包事务，自动管理Begin/Rollback/Commit生命周期
//
// 要求客户端传递完整数据（id、title、content、author_id）：
//   - id = 0：新建文章，INSERT 到制作库 + UPSERT 到线上库
//   - id > 0：已存在文章，UPDATE 制作库 + UPSERT 到线上库
func (ad *articleDAO) Sync(ctx context.Context, article Article) (int64, error) {
	var (
		id  = article.Id
		err error
	)
	err = ad.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txDAO := NewArticleDAO(tx)
		if id > 0 {
			err = txDAO.UpdateById(ctx, article)
		} else {
			id, err = txDAO.Insert(ctx, article)
			article.Id = id
		}
		if err != nil {
			return err
		}
		return txDAO.Upsert(ctx, PublishArticle(article))
	})
	return id, err
}

// Upsert 线上库插入或更新（UPSERT），实现INSERT OR UPDATE语义
// 使用GORM的OnConflict子句，对应MySQL的INSERT ... ON DUPLICATE KEY UPDATE
// 若主键冲突则更新标题、内容、状态和更新时间，否则插入新记录
func (ad *articleDAO) Upsert(ctx context.Context, article PublishArticle) error {
	now := time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := ad.db.Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]interface{}{
			"title":   article.Title,
			"content": article.Content,
			"utime":   now,
			"status":  article.Status,
		}),
	}).Create(&article).Error
	return err
}

// SyncStatus 事务内同步更新两库的文章状态，保证状态一致性
// 先更新制作库（带作者ID校验），再更新线上库
func (ad *articleDAO) SyncStatus(ctx context.Context, articleId, Uid int64, status domain.ArticleStatus) (int64, error) {
	var (
		id = articleId
	)
	now := time.Now().UnixMilli()
	return id, ad.db.Transaction(func(tx *gorm.DB) error {
		// 制作库：带作者ID校验，防止修改他人文章
		res := tx.Model(&Article{}).
			Where("id = ? AND author_id = ?", id, Uid).
			Updates(map[string]any{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("更新失败，可能是作者非法，id %d，author_id %d", articleId, Uid)
		}
		// 线上库：只按 id 更新（读者视角，无需校验作者）
		return tx.Model(&PublishArticle{}).
			Where("id = ? ", id).
			Updates(map[string]any{
				"status": status,
				"utime":  now,
			}).Error
	})
}

func (ad *articleDAO) GetByAuthor(ctx context.Context, authorId int64, offset int, limit int) ([]Article, error) {
	var articles []Article
	err := ad.db.WithContext(ctx).Model(&articles).
		Where("author_id = ?", authorId).
		Offset(offset).
		Limit(limit).
		Order("utime DESC").
		Find(&articles).Error
	return articles, err
}

func (ad *articleDAO) GetById(ctx context.Context, id int64) (Article, error) {
	var article Article
	err := ad.db.WithContext(ctx).Where("id = ?", id).First(&article).Error
	return article, err
}

func (ad *articleDAO) GetByPubId(ctx context.Context, id int64) (PublishArticle, error) {
	var pubArt PublishArticle
	err := ad.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, domain.ArticleStatusPublished.ToUint8()).
		First(&pubArt).Error
	return pubArt, err
}

// Article 文章数据库实体（制作库表）
type Article struct {
	Id      int64  `gorm:"primaryKey;autoIncrement"` // 主键ID
	Title   string `gorm:"type=varchar(1024)"`       // 文章标题
	Content string `gorm:"type=BLOB"`                // 文章内容（大文本）

	AuthorId int64 `gorm:"index"` // 作者ID（索引）
	Status   uint8 // 文章状态
	Ctime    int64 // 创建时间（毫秒时间戳）
	Utime    int64 // 更新时间（毫秒时间戳）
}

type PublishArticle Article

//type PublishArticle struct {
//	Article
//}
