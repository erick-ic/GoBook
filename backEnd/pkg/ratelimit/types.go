package ratelimit

import "context"

type Limiter interface {
	//Limited 是否触发限流，取值true，限流，key为限流对象
	//error 限流器本身错误
	Limited(ctx context.Context, key string) (bool, error)
}
