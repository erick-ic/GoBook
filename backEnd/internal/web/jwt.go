package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type jwtHandler struct{}

type UserClaims struct {
	//继承RegisteredClaims，实现claims
	jwt.RegisteredClaims
	//放入token的数据
	Uid       int64
	UserAgent string
}

func (jh jwtHandler) setJWTToken(ctx *gin.Context, uid int64) error {
	//生成token
	//token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
	//	"userId": u.Id,
	//})
	claims := UserClaims{
		//设置token有效期
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       uid,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key 95osj3fUD7fo0mlYdDbncXz4VD2igvf0
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		return err
	}
	//通过Http Response Header x-jwt-token返回
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}
