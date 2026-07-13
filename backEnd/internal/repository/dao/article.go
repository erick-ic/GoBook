package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ArticleDAO interface {
	Insert(ctx context.Context, article Article) (int64, error)
	Update(ctx context.Context, article Article) error
}

type articleDAO struct {
	db *gorm.DB
}

func NewArticleDAO(db *gorm.DB) ArticleDAO {
	return &articleDAO{
		db: db,
	}
}

func (ad articleDAO) Insert(ctx context.Context, article Article) (int64, error) {
	now := time.Now().UnixMilli()
	article.Ctime = now
	article.Utime = now
	err := ad.db.WithContext(ctx).Create(&article).Error
	return article.Id, err
}

func (ad articleDAO) Update(ctx context.Context, article Article) error {
	now := time.Now().UnixMilli()
	article.Utime = now

	//依赖gorm忽略零值的特性，会用主键进行更新。（可读性很差，不推荐）
	//return ad.db.WithContext(ctx).Updates(&article).Error

	res := ad.db.WithContext(ctx).Model(&article).
		//带上作者ID防止更改他人的帖子
		Where("id = ? AND author_id = ?", article.Id, article.AuthorId).
		Updates(map[string]any{
			"title":   article.Title,
			"content": article.Content,
			"utime":   article.Utime,
		})

	//检查是否修改成功
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		//记录日志
		return fmt.Errorf("更新失败，可能是作者非法，id %d，author_id %d", article.Id, article.AuthorId)
	}
	return res.Error
}

type Article struct {
	Id      int64  `gorm:"primaryKey;autoIncrement"`
	Title   string `gorm:"type=varchar(1024)"`
	Content string `gorm:"type=BLOB"`

	//如何设置索引？

	AuthorId int64 `gorm:"index"`
	Ctime    int64
	Utime    int64
}
