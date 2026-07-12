package ioc

import (
	"GoBook/config"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	addr := viper.GetString("redis.addr")
	fmt.Println("redis addr:", addr)

	redisClient := redis.NewClient(&redis.Options{
		//Addr: "localhost:6379",
		//Addr: "gobook-redis:11479",
		Addr: config.Config.Redis.Addr,
	})
	return redisClient
}
