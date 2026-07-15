package article

import (
	"context"

	"gorm.io/gorm"
)

// AuthorDAO 制作库数据访问对象接口，定义制作库的文章操作
type AuthorDAO interface {
	Insert(ctx context.Context, article Article) (int64, error) // 在制作库插入文章
	UpdateById(ctx context.Context, article Article) error      // 在制作库更新文章
}

// authorDAO 制作库数据访问对象实现类（预留，尚未实现）
type authorDAO struct {
	db *gorm.DB // 数据库连接
}

// NewAuthorDAO 创建制作库数据访问对象实例
func NewAuthorDAO(db *gorm.DB) AuthorDAO {
	return &authorDAO{
		db: db,
	}
}

// Insert 在制作库插入文章记录（预留接口，尚未实现）
func (a authorDAO) Insert(ctx context.Context, article Article) (int64, error) {
	panic("implement me")
}

// UpdateById 在制作库更新文章记录（预留接口，尚未实现）
func (a authorDAO) UpdateById(ctx context.Context, article Article) error {
	panic("implement me")
}
