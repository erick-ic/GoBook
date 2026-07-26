package main

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

func main() {
	initViper()

	app := InitApp()

	for _, c := range app.consumers {
		err := c.Start()
		if err != nil {
			panic(err)
		}
	}

	if err := app.server.Serve(); err != nil {
		log.Println(err)
	}
}

// initViper 加载互动微服务配置。
// 同时支持从 backEnd 目录执行 go run ./interactive，
// 以及进入 backEnd/interactive 后直接执行 go run .。
func initViper() {
	viper.SetConfigName("dev")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("./interactive/config")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
