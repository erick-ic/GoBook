package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/cache"
	"GoBook/internal/repository/dao"
	"context"
)

type InteractiveRepository interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	IncrLike(ctx context.Context, biz string, id int64, uid int64) error
	DecrLike(ctx context.Context, biz string, id int64, uid int64) error
	Get(ctx context.Context, biz string, id int64) (domain.Interactive, error)
}

type interactiveRepository struct {
	dao   dao.InteractiveDAO
	cache cache.InteractiveCache
}

func (ir *interactiveRepository) Get(ctx context.Context, biz string, id int64) (domain.Interactive, error) {
	inter, err := ir.dao.Get(ctx, biz, id)
	if err != nil {
		return domain.Interactive{}, err
	}
	res := ir.toDomain(inter)
	return res, nil
}

func (ir *interactiveRepository) IncrLike(ctx context.Context, biz string, id int64, uid int64) error {
	err := ir.dao.InsertLikeInfo(ctx, biz, id, uid)
	if err != nil {
		return err
	}
	return ir.cache.IncrLikeCntIfPresent(ctx, biz, id)
}

func (ir *interactiveRepository) DecrLike(ctx context.Context, biz string, id int64, uid int64) error {
	err := ir.dao.DeleteLikeInfo(ctx, biz, id, uid)
	if err != nil {
		return err
	}
	return nil
}

func (ir *interactiveRepository) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	err := ir.dao.IncrReadCnt(ctx, biz, bizId)
	if err != nil {
		return err
	}
	//缓存方案
	//优先保证数据库的准确性，先走数据库
	return ir.cache.IncrReadCntIfPresent(ctx, biz, bizId)
}

func NewInteractiveRepository(dao dao.InteractiveDAO, cache cache.InteractiveCache) InteractiveRepository {
	return &interactiveRepository{
		dao:   dao,
		cache: cache,
	}
}

func (ir *interactiveRepository) toDomain(ie dao.Interactive) domain.Interactive {
	return domain.Interactive{
		ReadCnt:    ie.ReadCnt,
		LikeCnt:    ie.LikeCnt,
		CollectCnt: ie.CollectCnt,
	}
}
