package middleware

import (
	"net/http"
	"time"

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
		//if ctx.Request.URL.Path == "/users/login" || ctx.Request.URL.Path == "/users/signup" {
		//	return
		//}
		for _, path := range lb.paths {
			if ctx.Request.URL.Path == path {
				return
			}
		}

		sess := sessions.Default(ctx)
		userId := sess.Get("userId")

		if userId == nil {
			//未登录
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		//登录后，刷新cookie
		updateTime := sess.Get("updateTime")
		sess.Set("userId", userId)
		//整个会话的有效期
		sess.Options(sessions.Options{
			MaxAge: 30,
		})
		nowTime := time.Now().UnixMilli()

		//不存在updateTime，未刷新过，即首次登录
		if updateTime == nil {
			sess.Set("updateTime", nowTime)
			sess.Save()
			return
		}

		//存在updateTime, 需要再次刷新
		updateTimeVal, _ := updateTime.(int64)

		//时间大于1分钟
		//if nowTime-updateTimeVal > 60*1000 {
		//	sess.Set("updateTime", nowTime)
		//}
		if nowTime-updateTimeVal > 10*1000 {
			//超过阈值，刷新时间，保存 session（即续期）
			sess.Set("updateTime", nowTime)
			sess.Save()
			return
		}
		//未超过阈值，不做任何保存，保持现有有效期
		//若长时间未发生后续请求，超出有效期，userId、updateTime会被删除，当发生请求时，触发去登录
	}
}
