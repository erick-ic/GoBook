package dao

import (
	"gorm.io/gorm"
)

func InitTable(db *gorm.DB) error {
	return db.AutoMigrate(
		&UserLikeBiz{},
		// 用户收藏明细表；聚合收藏数仍保存在 Interactive 表中。
		&UserCollectionBiz{},
		&Interactive{},
	)
}
