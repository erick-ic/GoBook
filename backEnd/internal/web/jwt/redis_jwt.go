package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	AccessTokenKey  = []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0")
	RefreshTokenKey = []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf1")
)

type RedisJWTHandler struct {
	cmd redis.Cmdable
}

func NewRedisJWTHandler(cmd redis.Cmdable) JWTHandler {
	return &RedisJWTHandler{
		cmd: cmd,
	}
}

type UserClaims struct {
	//继承RegisteredClaims，实现claims
	jwt.RegisteredClaims
	//放入token的数据
	Uid       int64
	UserAgent string
	Ssid      string
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Uid  int64
	Ssid string
}

func (rj *RedisJWTHandler) SetLoginToken(ctx *gin.Context, uid int64) error {
	ssid := uuid.New().String()
	err := rj.SetJWTToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	err = rj.setRefreshToken(ctx, uid, ssid)
	if err != nil {
		return err
	}
	return nil
}

func (rj *RedisJWTHandler) SetJWTToken(ctx *gin.Context, uid int64, Ssid string) error {
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
		Ssid:      Ssid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key 95osj3fUD7fo0mlYdDbncXz4VD2igvf0
	tokenStr, err := token.SignedString(AccessTokenKey)
	if err != nil {
		return err
	}
	//通过Http Response Header x-jwt-token返回
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

func (rj *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	ctx.Header("X-JWT-Token", "")
	ctx.Header("x-refresh-token", "")

	//val := ctx.MustGet("claims")
	//claims, _ := val.(*UserClaims)
	claims := ctx.MustGet("claims").(*UserClaims)
	//标记ssid
	return rj.cmd.Set(ctx, fmt.Sprintf("users:ssid:%s", claims.Ssid),
		"", time.Hour*24*7).Err()
}

func (rj *RedisJWTHandler) CheckSession(ctx *gin.Context, Ssid string) error {
	//查看ssid是否有效
	_, err := rj.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", Ssid)).Result()
	return err
}

func (rj *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	//读取前端请求头中的Authorization
	headerToken := ctx.GetHeader("Authorization")

	//Bearer eyJhbGciOiJIUzI1N...
	segs := strings.Split(headerToken, " ")
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}

func (rj *RedisJWTHandler) setRefreshToken(ctx *gin.Context, uid int64, Ssid string) error {
	claims := RefreshClaims{
		//设置token有效期
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
		Uid:  uid,
		Ssid: Ssid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key
	tokenStr, err := token.SignedString(RefreshTokenKey)
	if err != nil {
		return err
	}
	//通过Http Response Header x-refresh-token返回
	ctx.Header("x-refresh-token", tokenStr)
	return nil
}
