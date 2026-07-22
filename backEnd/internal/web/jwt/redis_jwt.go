package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrSessionRevoked = errors.New("登录会话已注销")
	ErrInvalidClaims  = errors.New("JWT claims 无效")
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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 60)),
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

// ClearToken 注销当前会话：清空返回给客户端的令牌，并将当前 SSID 写入 Redis 黑名单。
// 黑名单记录保留 7 天，与刷新令牌的有效期一致，防止已注销的令牌在过期前再次使用。
func (rj *RedisJWTHandler) ClearToken(ctx *gin.Context) error {
	// 清空响应头中的访问令牌和刷新令牌，通知客户端删除本地保存的令牌。
	ctx.Header("X-JWT-Token", "")
	ctx.Header("x-refresh-token", "")

	// 认证中间件应提前校验 JWT，并将 *UserClaims 保存到 Gin 上下文的 claims 中。
	// 使用安全读取代替 MustGet，避免路由漏挂中间件时因 claims 不存在而触发 panic。
	val, exists := ctx.Get("claims")
	if !exists {
		return fmt.Errorf("%w: 上下文中不存在 claims", ErrInvalidClaims)
	}

	// 安全断言 claims 类型，避免上下文数据类型错误时触发 panic。
	claims, ok := val.(*UserClaims)
	if !ok || claims == nil {
		return fmt.Errorf("%w: claims 类型不是 *UserClaims", ErrInvalidClaims)
	}
	// SSID 是会话黑名单键的一部分，为空时不能生成有效的注销记录。
	if claims.Ssid == "" {
		return fmt.Errorf("%w: SSID 为空", ErrInvalidClaims)
	}

	// 以 SSID 为键写入注销黑名单；CheckSession 会据此拒绝该会话后续的请求。
	// Redis 写入失败时将错误交给上层处理。
	/*
		ctx：传递请求的取消和超时信号；
		key：users:ssid:<ssid>，唯一标识当前登录会话；
		value：空字符串，因为这里只需要判断 Key 是否存在；
		expiration：7 天，与刷新令牌的有效期保持一致。
	*/
	return rj.cmd.Set(ctx, fmt.Sprintf("users:ssid:%s", claims.Ssid),
		"", time.Hour*24*7).Err()
}

func (rj *RedisJWTHandler) CheckSession(ctx *gin.Context, Ssid string) error {
	// users:ssid:<ssid> 是注销黑名单：key 存在表示该会话已经退出登录。
	cnt, err := rj.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", Ssid)).Result()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return ErrSessionRevoked
	}
	return nil
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
