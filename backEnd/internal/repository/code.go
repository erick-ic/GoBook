package repository

import (
	"GoBook/internal/repository/cache"
	"context"
)

var (
	ErrCodeSendTooMany        = cache.ErrCodeSendTooMany
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
)

type CodeRepository interface {
	Store(ctx context.Context, biz, phone, code string) error
	VerifyCode(ctx context.Context, biz, phone, code string) (bool, error)
}

type CacheCodeRepository struct {
	cache cache.CodeCache
}

func NewCodeRepository(cache cache.CodeCache) CodeRepository {
	return &CacheCodeRepository{
		cache: cache,
	}
}

func (cr *CacheCodeRepository) Store(ctx context.Context, biz, phone, code string) error {
	return cr.cache.SetCode(ctx, biz, phone, code)
}

func (cr *CacheCodeRepository) VerifyCode(ctx context.Context, biz, phone, code string) (bool, error) {
	return cr.cache.VerifyCode(ctx, biz, phone, code)
}
