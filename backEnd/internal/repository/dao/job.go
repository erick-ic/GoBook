package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type JobDAO interface {
	// Preempt 从到期任务中抢占一条，并把它从 waiting 切换为 running。
	Preempt(ctx context.Context) (Job, error)
	// Release 将执行完毕或失败的任务重新置为 waiting。
	Release(ctx context.Context, jid int64) error
	// UpdateTime 更新运行中任务的心跳时间。
	UpdateTime(ctx context.Context, id int64) error
	// UpdateNextTime 保存任务按 Cron 表达式计算出的下次执行时间。
	UpdateNextTime(ctx context.Context, id int64, t time.Time) error
}

// GORMJobDAO 使用 MySQL 任务表作为多实例之间共享的调度状态。
type GORMJobDAO struct {
	db *gorm.DB
}

// UpdateNextTime 仅更新下一次执行时间；任务状态随后由 CancelFunc 调用 Release 恢复。
func (gj *GORMJobDAO) UpdateNextTime(ctx context.Context, id int64, t time.Time) error {
	now := time.Now().UnixMilli()
	return gj.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", id).Updates(map[string]any{
		"utime":     now,
		"next_time": t.UnixMilli(),
	}).Error
}

// UpdateTime 写入当前时间作为心跳，表明持有任务的实例仍在正常运行。
func (gj *GORMJobDAO) UpdateTime(ctx context.Context, id int64) error {
	now := time.Now().UnixMilli()
	return gj.db.WithContext(ctx).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"utime": now,
		}).Error
}

// Release 结束本轮占用，使任务可在 next_time 再次到期后被其他实例抢占。
func (gj *GORMJobDAO) Release(ctx context.Context, jid int64) error {
	now := time.Now().UnixMilli()
	return gj.db.WithContext(ctx).Model(&Job{}).
		Where("id = ?", jid).
		Updates(map[string]interface{}{
			"status": jobStatusWaiting,
			"utime":  now,
		}).Error
}

/*
高并发优化方案：分布式任务调度系统
方案1.一次拉一批，如一次取100条，随机从某一条开始向后抢占
方案2.随机偏移量，0-100生成随机偏移量，第一轮若没抢到则偏移量回归0
方案3.id取余分配，如status=? AND next_time<=? AND id%10=?,若没找到，则不加余数条件重试
*/
// Preempt 使用“先查询候选任务，再通过 version 做 CAS 更新”的方式抢占任务：
//  1. 找到一条 waiting 且已经到期的任务；
//  2. 以 id+version 为条件把状态切换为 running，并递增 version；
//  3. RowsAffected 为 0 表示已被其他实例抢走，继续查找下一轮候选任务。
func (gj *GORMJobDAO) Preempt(ctx context.Context) (Job, error) {
	for {
		var j Job
		now := time.Now().UnixMilli()
		err := gj.db.Where("status = ? AND next_time < ?", jobStatusWaiting, now).First(&j).Error
		if err != nil {
			return Job{}, err
		}

		//乐观锁 CAS compare and swap
		//面试常见：用乐观锁取代for update，因为for update易产生死锁问题。
		res := gj.db.WithContext(ctx).Model(&Job{}).
			Where("id = ? AND version = ?", j.Id, j.Version).
			Updates(map[string]interface{}{
				"status":  jobStatusRunning,
				"version": j.Version + 1,
				"utime":   now,
			})
		if res.Error != nil {
			return Job{}, res.Error
		}
		if res.RowsAffected == 0 {
			//没抢到，继续下一轮
			continue
		}

		return j, nil
	}
}

func NewGORMJobDAO(db *gorm.DB) JobDAO {
	return &GORMJobDAO{db: db}
}

// Job 是数据库中的任务定义和调度状态。
// 与 domain.Job 相比，它额外保存抢占状态、乐观锁版本和下一次执行时间。
type Job struct {
	Id         int64  `gorm:"primaryKey,autoIncrement"`
	Name       string `gorm:"type:varchar(128);unique"`
	Executor   string
	Expression string
	Cfg        string
	// Status 表示任务是否可抢占；Version 用于多实例之间的乐观锁竞争。
	Status int

	Version int

	// NextTime、Utime、Ctime 均以 Unix 毫秒保存。
	// NextTime 参与到期任务查询，Utime 同时承担运行中心跳时间的作用。
	NextTime int64 `gorm:"index"`

	Utime int64
	Ctime int64
}

const (
	// jobStatusWaiting 表示当前没有实例持有，可在到期后参与抢占。
	jobStatusWaiting = iota
	// jobStatusRunning 表示已经被某个调度实例抢占并正在执行。
	jobStatusRunning
	// jobStatusPaused 表示任务被暂停，不再参与调度。
	jobStatusPaused
)
