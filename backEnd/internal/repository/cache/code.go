package cache

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var (
	ErrCodeSendTooMany        = errors.New("发送过于频繁")
	ErrCodeVerifyTooManyTimes = errors.New("验证次数过多")
	ErrUnknowForCode          = errors.New("验证码未知错误")
)

// 注入lua脚本
// 编译器会在编译的时候，把set_code放入luaSetCode

//go:embed lua/send_code.lua
var luaSetCode string

//go:embed lua/verify_code.lua
var luaVerifyCode string

type CodeCache interface {
	SetCode(ctx context.Context, biz, phone, code string) error
	VerifyCode(ctx context.Context, biz, phone, inputCode string) (bool, error)
}
type RedisCodeCache struct {
	client redis.Cmdable
}

func NewCodeCache(client redis.Cmdable) CodeCache {
	return &RedisCodeCache{
		client: client,
	}
}

func (cc *RedisCodeCache) SetCode(ctx context.Context, biz, phone, code string) error {
	/*
		cc.client.Eval(...) 执行 Lua 脚本，返回 *redis.Cmd 对象（一个命令结果容器）。
			- ctx：上下文，用于超时控制或链路追踪（若 Redis 操作超过上下文 Deadline 会取消）。
			- luaSetCode：通过 //go:embed 嵌入的 Lua 脚本源码字符串
			- []string{cc.key(biz, phone)}：KEYS 列表。在 Lua 脚本中通过 KEYS[1] 访问。这里传入一个只包含一个元素的切片，即类似 "phone_code:login:18715156541" 的字符串。
			- code：ARGV 列表。在 Lua 脚本中通过 ARGV[1] 访问。这里传入生成的验证码（如 "080826"）。（注意：虽然这里只显式传了一个 code，但客户端实际会将其打包成可变参数 ...interface{}，对应 Lua 中的 ARGV[1]）
	*/

	//1.将 Redis Key（如 phone_code:login:18715156541）传给 Lua 的 KEYS[1]。
	//2.将验证码code（如 "080826"）传给 Lua 的 ARGV[1]。
	//3.脚本执行完毕后，返回状态码（0、-1 或 -2），通过 .Int() 转换为 res。
	res, err := cc.client.Eval(ctx, luaSetCode, []string{cc.key(biz, phone)}, code).Int()
	if err != nil {
		return err
	}
	switch res {
	case 0:
		//一切正常
		return nil
	case -1:
		return ErrCodeSendTooMany
	default:
		return errors.New("系统错误！")
	}
}

func (cc *RedisCodeCache) VerifyCode(ctx context.Context, biz, phone, inputCode string) (bool, error) {
	res, err := cc.client.Eval(ctx, luaVerifyCode, []string{cc.key(biz, phone)}, inputCode).Int()
	if err != nil {
		return false, err
	}
	switch res {
	case 0:
		return true, nil
	case -1:
		return false, ErrCodeVerifyTooManyTimes
	case -2:
		return false, nil
	default:
		return false, ErrUnknowForCode
	}
}

func (cc *RedisCodeCache) key(biz, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
