package repository

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/dao"
	"context"
	"time"
)

type JobRepository interface {
	Preempt(ctx context.Context) (domain.Job, error)
	Release(ctx context.Context, jid int64) error
	UpdateTime(ctx context.Context, id int64) error
	UpdateNextTime(ctx context.Context, id int64, time time.Time) error
}

// jobRepository 负责隔离 DAO 数据模型与调度层的 domain.Job。
type jobRepository struct {
	dao dao.JobDAO
}

// UpdateNextTime 将 Service 计算出的下次触发时间持久化。
func (jr *jobRepository) UpdateNextTime(ctx context.Context, id int64, time time.Time) error {
	return jr.dao.UpdateNextTime(ctx, id, time)
}

// UpdateTime 透传任务心跳更新时间。
func (jr *jobRepository) UpdateTime(ctx context.Context, id int64) error {
	return jr.dao.UpdateTime(ctx, id)
}

// Release 透传任务释放操作，将运行中任务恢复为可调度状态。
func (jr *jobRepository) Release(ctx context.Context, jid int64) error {
	return jr.dao.Release(ctx, jid)
}

// Preempt 完成数据库抢占，并丢弃只属于持久化层的 status、version 等字段。
// 当前仅映射 Id、Expression、Executor、Name，DAO 中的 Cfg 尚未透传到 domain.Job。
func (jr *jobRepository) Preempt(ctx context.Context) (domain.Job, error) {
	res, err := jr.dao.Preempt(ctx)
	return domain.Job{
		Id:         res.Id,
		Expression: res.Expression,
		Executor:   res.Executor,
		Name:       res.Name,
	}, err
}

func NewJobRepository(dao dao.JobDAO) JobRepository {
	return &jobRepository{
		dao: dao,
	}
}
