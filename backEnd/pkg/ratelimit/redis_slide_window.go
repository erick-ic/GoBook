package ratelimit

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed slide_window.lua
var luaSlideWindow string

// RedisSlideWindowLimiter Redis上的滑动窗口算法限流器实现
type RedisSlideWindowLimiter struct {
	cmd redis.Cmdable

	/*
		interval内允许rate个请求
	*/
	//窗口大小
	interval time.Duration
	// 阈值
	rate int
}

func NewRedisSlideWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) Limiter {
	return &RedisSlideWindowLimiter{
		cmd:      cmd,
		interval: interval,
		rate:     rate,
	}
}

func (r *RedisSlideWindowLimiter) Limited(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, luaSlideWindow, []string{key},
		r.interval.Milliseconds(),
		r.rate, time.Now().UnixMilli()).Bool()
}
