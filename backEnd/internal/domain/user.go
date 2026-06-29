package domain

// User 领域对象，用户的抽象 DDD中的entity
// 别称：BO(business object)
type User struct {
	Email    string
	Password string
}
