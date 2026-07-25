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
		// 用户收藏明细表；聚合收藏数仍保存在 Interactive 表中。
		&UserCollectionBiz{},
		&Interactive{},
		// Job 保存分布式调度任务的定义、抢占状态与下一次执行时间。
		&Job{},
	)
}
