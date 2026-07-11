package web

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type jwtHandler struct {
	accessTokenKey  []byte
	refreshTokenKey []byte
}

func newJwtHandler() jwtHandler {
	return jwtHandler{
		accessTokenKey:  []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"),
		refreshTokenKey: []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf1"),
	}
}

type UserClaims struct {
	//继承RegisteredClaims，实现claims
	jwt.RegisteredClaims
	//放入token的数据
	Uid       int64
	UserAgent string
}

func (jh *jwtHandler) setJWTToken(ctx *gin.Context, uid int64) error {
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
	tokenStr, err := token.SignedString(jh.accessTokenKey)
	if err != nil {
		return err
	}
	//通过Http Response Header x-jwt-token返回
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid int64
}

func (jh *jwtHandler) setRefreshToken(ctx *gin.Context, uid int64) error {
	claims := RefreshClaims{
		//设置token有效期
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
		Uid: uid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key
	tokenStr, err := token.SignedString(jh.refreshTokenKey)
	if err != nil {
		return err
	}
	//通过Http Response Header x-refresh-token返回
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}

func ExtractToken(ctx *gin.Context) string {
	//读取前端请求头中的Authorization
	headerToken := ctx.GetHeader("Authorization")

	//Bearer eyJhbGciOiJIUzI1N...
	segs := strings.Split(headerToken, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}
