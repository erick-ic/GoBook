//go:build !k8s

// 没有k8s编译标签
package config

var Config = config{
	DB: DBConfig{
		//本地连接
		DSN: "localhost:13316",
	},
	Redis: RedisConfig{
		//本地连接
		Addr: "localhost:6379",
	},
}
