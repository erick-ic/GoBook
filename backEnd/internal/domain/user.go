package domain

import "time"

// User 领域对象，用户的抽象 DDD中的entity
// 别称：BO(business object)
type User struct {
	Id       int64
	Email    string
	Password string
	Phone    string
	Ctime    time.Time
}
