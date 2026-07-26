package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// InitRedis 根据 redis.addr 创建互动服务使用的 Redis 客户端。
func InitRedis() redis.Cmdable {
	addr := viper.GetString("redis.addr")
	if addr == "" {
		panic("interactive: redis.addr 不能为空")
	}

	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
