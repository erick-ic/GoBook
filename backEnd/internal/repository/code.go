package repository

import (
	"GoBook/internal/repository/cache"
	"context"
)

var (
	ErrCodeSendTooMany        = cache.ErrCodeSendTooMany
	ErrCodeVerifyTooManyTimes = cache.ErrCodeVerifyTooManyTimes
)

type CodeRepository struct {
	cache *cache.CodeCache
}

func NewCodeRepository(cache *cache.CodeCache) *CodeRepository {
	return &CodeRepository{
		cache: cache,
	}
}

func (cr *CodeRepository) Store(ctx context.Context, biz, phone, code string) error {
	return cr.cache.SetCode(ctx, biz, phone, code)
}

func (cr *CodeRepository) VerifyCode(ctx context.Context, biz, phone, code string) (bool, error) {
	return cr.cache.VerifyCode(ctx, biz, phone, code)
}
