//go:build k8s

// 添加k8s标签
package config

var Config = config{
	DB: DBConfig{
		DSN: "root:root@tcp(gobook-mysql:11309)/gobook?charset=utf8mb4&parseTime=True&loc=Local",
	},
	Redis: RedisConfig{
		Addr: "gobook-redis:11479",
	},
}
