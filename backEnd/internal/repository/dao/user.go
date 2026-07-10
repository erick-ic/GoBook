package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserDuplicated = errors.New("邮箱/手机号冲突！")
	ErrUserNotFound   = gorm.ErrRecordNotFound
)

type UserDAO interface {
	FindByPhone(ctx context.Context, phone string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindById(ctx context.Context, id int64) (User, error)
	FindByWechat(ctx context.Context, openId string) (User, error)
	Insert(ctx context.Context, u User) error
}
type GORMUserDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &GORMUserDAO{
		db: db,
	}
}

func (ud *GORMUserDAO) FindByWechat(ctx context.Context, openId string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).Where("wechat_open_id = ?", openId).First(&u).Error
	return u, err
}

func (ud *GORMUserDAO) FindByPhone(ctx context.Context, phone string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).Where("phone = ?", phone).First(&u).Error
	return u, err
}

func (ud *GORMUserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	return u, err
}

func (ud *GORMUserDAO) FindById(ctx context.Context, id int64) (User, error) {
	var u User
	err := ud.db.WithContext(ctx).Where("`id` = ?", id).First(&u).Error
	return u, err
}

func (ud *GORMUserDAO) Insert(ctx context.Context, u User) error {
	//存入毫秒数
	now := time.Now().UnixMilli()
	u.Ctime = now
	u.Utime = now
	err := ud.db.WithContext(ctx).Create(&u).Error

	// 捕获 MySQL 1062 错误（唯一键冲突）
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictsErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictsErrNo {
			//唯一索引冲突，即邮箱/手机号冲突
			return ErrUserDuplicated
		}
	}
	return err
}

// User 数据库表结构
// 别称entity、model、PO(persistent object)
type User struct {
	Id       int            `gorm:"primaryKey, autoIncrement"`
	Email    sql.NullString `gorm:"unique"`
	Password string
	//唯一索引允许有多个空值，但不能有多个""
	//Phone *string        //早期写法需要解引流，判空
	Phone sql.NullString `gorm:"unique"` //唯一索引，允许 NULL

	//微信字段 wechatInfo
	WechatUnionId sql.NullString
	WechatOpenId  sql.NullString `gorm:"unique"`

	//创建时间，毫秒数
	Ctime int64
	//更新时间，毫秒数
	Utime int64
}
