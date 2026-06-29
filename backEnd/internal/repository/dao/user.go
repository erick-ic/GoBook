package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type UserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{
		db: db,
	}
}

func (ud *UserDAO) Insert(ctx context.Context, u User) error {
	//存入毫秒数
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	return ud.db.WithContext(ctx).Create(&u).Error
}

// User 数据库表结构
// 别称entity、model、PO(persistent object)
type User struct {
	Id       int    `gorm:"primaryKey, autoIncrement"`
	Email    string `gorm:"unique"`
	Password string

	//创建时间，毫秒数
	Ctime int64
	//更新时间，毫秒数
	Utime int64
}
