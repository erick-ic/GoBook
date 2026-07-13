package startup

import (
	"GoBook/config"

	"github.com/redis/go-redis/v9"
)

func InitRedis() redis.Cmdable {
	redisClient := redis.NewClient(&redis.Options{
		//Addr: "localhost:6379",
		//Addr: "gobook-redis:11479",
		Addr: config.Config.Redis.Addr,
	})
	return redisClient
}
