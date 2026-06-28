package web

import "GoBook/internal/web/user"

import (
	"github.com/gin-gonic/gin"
)

// RegisterRouters 注册路由
func RegisterRouters() *gin.Engine {
	server := gin.Default()

	RegisterUsersRouters(server)

	return server
}

// RegisterUsersRouters 注册user相关路由
func RegisterUsersRouters(server *gin.Engine) {
	//u := &user.UserHandler{}
	//server.POST("v1/users/signUp", u.SignUp)
	//server.POST("v1/users/login", u.Login)
	//server.POST("v1/users/create", u.Create)
	//server.POST("v1/users/delete", u.Delete)
	//server.POST("v1/users/edit", u.Edit)
	//server.GET("v1/users/profile", u.Profile)

	u := user.NewUserHandler()

	//路由分组
	ug := server.Group("v1/users")
	ug.POST("/signUp", u.SignUp)
	ug.POST("/login", u.Login)
	ug.POST("/create", u.Create)
	ug.POST("/delete", u.Delete)
	ug.POST("/edit", u.Edit)
	ug.GET("/profile", u.Profile)
}
