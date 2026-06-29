package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type LoginMiddlewareBuilder struct {
	paths []string
}

func NewLoginMiddlewareBuilder() *LoginMiddlewareBuilder {
	return &LoginMiddlewareBuilder{}
}

func (lb *LoginMiddlewareBuilder) IgnoreLoginMiddlewareBuilder(path string) *LoginMiddlewareBuilder {
	lb.paths = append(lb.paths, path)
	return lb

}
func (lb *LoginMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//无需登录校验接口
		//if ctx.Request.URL.Path == "/v1/users/login" || ctx.Request.URL.Path == "/v1/users/signup" {
		//	return
		//}
		for _, path := range lb.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}
		sess := sessions.Default(ctx)
		id := sess.Get("userId")
		if id == nil {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}
