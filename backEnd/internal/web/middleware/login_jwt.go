package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginJWTMiddlewareBuilder struct {
	paths []string
}

func NewLoginJWTMiddlewareBuilder() *LoginJWTMiddlewareBuilder {
	return &LoginJWTMiddlewareBuilder{}
}

func (ljb *LoginJWTMiddlewareBuilder) IgnorePaths(path string) *LoginJWTMiddlewareBuilder {
	ljb.paths = append(ljb.paths, path)
	return ljb

}
func (ljb *LoginJWTMiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//无需登录校验接口
		//if ctx.Request.URL.Path == "/users/login" || ctx.Request.URL.Path == "/users/signup" {
		//	return
		//}
		for _, path := range ljb.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}

		//使用JWT校验
		headerToken := ctx.GetHeader("Authorization")
		if headerToken == "" {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		segs := strings.Split(headerToken, " ")
		fmt.Println(segs)
		if len(segs) != 2 {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		tokenStr := segs[1]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"), nil
		})
		if err != nil {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if token == nil || !token.Valid {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	}
}
