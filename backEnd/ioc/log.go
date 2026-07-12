package ioc

import "go.uber.org/zap"

// InitLogger 全局Logger
func InitLogger() *zap.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return l
}
