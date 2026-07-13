package startup

import (
	"GoBook/internal/repository/dao"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {

	db, err := gorm.Open(mysql.Open("root:root@tcp(localhost:13316)/gobook?charset=utf8mb4&parseTime=True&loc=Local"))
	if err != nil {
		//一旦初始化过程报错，应用就取消启动
		//panic相当于整个goroutine结束
		panic(err)
	}

	//自动初始化表
	err = dao.InitTable(db)
	if err != nil {
		panic(err)
	}
	return db
}
