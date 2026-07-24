package dao

import (
	"GoBook/internal/repository/dao/article"

	"gorm.io/gorm"
)

func InitTable(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&article.Article{},
		&article.PublishArticle{},
		&UserLikeBiz{},
		&Interactive{},
		// Job 保存分布式调度任务的定义、抢占状态与下一次执行时间。
		&Job{},
	)
}
