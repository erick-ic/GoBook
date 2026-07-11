package middleware

import (
	"GoBook/internal/web"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type LoginJWTMiddlewareBuilder struct {
	paths []string
	cmd   redis.Cmdable
}

func NewLoginJWTMiddlewareBuilder(cmd redis.Cmdable) *LoginJWTMiddlewareBuilder {
	return &LoginJWTMiddlewareBuilder{
		cmd: cmd,
	}
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
		tokenStr := web.ExtractToken(ctx)
		claims := &web.UserClaims{}
		//ParseWithClaims需要传入指针
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"), nil
		})
		if err != nil {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if token == nil || !token.Valid || claims.Uid == 0 {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//检验UserAgent
		if claims.UserAgent != ctx.Request.UserAgent() {
			//严重的安全问题，使用了不同的设备环境
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//刷新token
		////有效期1分钟，目前不足50秒，刷新token，即每10秒刷新一次
		//if claims.ExpiresAt.Sub(time.Now()) < time.Second*50 {
		//	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Minute))
		//	tokenStr, err = token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
		//	if err != nil {
		//		//记录日志
		//		log.Println("jwt 续约失败:", err)
		//	}
		//	ctx.Header("x-jwt-token", tokenStr)
		//}

		//查看ssid是否有效
		cnt, err := ljb.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", claims.Ssid)).Result()
		if err != nil || cnt > 0 {
			//Redis问题或已退出登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//可在ctx中传递数据，进行读写操作。
		ctx.Set("claims", claims)
		ctx.Set("userId", claims.Uid)
	}
}
