package article

import (
	"context"

	"gorm.io/gorm"
)

// ReaderDAO 线上库数据访问对象接口，定义线上库的文章操作
type ReaderDAO interface {
	Upsert(ctx context.Context, article Article) error // 线上库插入或更新文章
}

// readerDAO 线上库数据访问对象实现类（预留，尚未实现）
type readerDAO struct {
	db *gorm.DB // 数据库连接
}

// NewReaderDAO 创建线上库数据访问对象实例
func NewReaderDAO(db *gorm.DB) ReaderDAO {
	return &readerDAO{
		db: db,
	}
}

// Upsert 线上库插入或更新文章（预留接口，尚未实现）
func (r readerDAO) Upsert(ctx context.Context, article Article) error {
	panic("implement me")
}

// PublishArticle 线上表实体，嵌套Article结构体，用于读者访问的已发布文章
type PublishArticle struct {
	Article
}
