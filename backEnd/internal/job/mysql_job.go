package job

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	"GoBook/pkg/logger"
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
)

// Executor 将调度框架与具体任务执行方式解耦。
// Scheduler 根据 domain.Job.Executor 找到同名实现并调用 Exec。
type Executor interface {
	Name() string
	// Exec 必须正确响应 ctx 的取消信号，确保 Scheduler 停止时任务能够退出。
	Exec(ctx context.Context, j domain.Job) error
}

// LocalFuncExecutor 以任务名称为 key，把数据库任务映射到进程内注册的 Go 函数。
type LocalFuncExecutor struct {
	funcs map[string]func(ctx context.Context, j domain.Job) error
}

func NewLocalFuncExecutor() *LocalFuncExecutor {
	return &LocalFuncExecutor{funcs: map[string]func(ctx context.Context, j domain.Job) error{}}
}

func (l *LocalFuncExecutor) Name() string {
	return "local"
}

// RegisterFunc 应在 Scheduler 启动前完成，避免执行过程中并发修改 funcs。
func (l *LocalFuncExecutor) RegisterFunc(name string, fn func(ctx context.Context, j domain.Job) error) {
	l.funcs[name] = fn
}

// Exec 根据任务名称寻找本地函数；未注册的任务不会被静默忽略。
func (l *LocalFuncExecutor) Exec(ctx context.Context, j domain.Job) error {
	fn, ok := l.funcs[j.Name]
	if !ok {
		return fmt.Errorf("未注册本地方法 %s", j.Name)
	}
	return fn(ctx, j)
}

// Scheduler 持续从数据库抢占到期任务，并把任务分派给已注册的 Executor。
type Scheduler struct {
	// dbTimeout 限制单次抢占数据库的时间，避免数据库异常阻塞整个调度循环。
	dbTimeout time.Duration

	svc service.JobService

	// executors 按执行器名称索引；limiter 控制本实例同时持有和执行的任务数量。
	executors map[string]Executor
	l         logger.LoggerV1

	limiter *semaphore.Weighted
}

// NewScheduler 默认允许当前实例最多并发执行 100 个任务。
func NewScheduler(svc service.JobService, l logger.LoggerV1) *Scheduler {
	return &Scheduler{
		svc:       svc,
		dbTimeout: time.Second,
		limiter:   semaphore.NewWeighted(100),
		l:         l,
		executors: map[string]Executor{},
	}
}

// RegisterExecutor 注册一种任务执行方式，应在调用 Schedule 前完成。
func (s *Scheduler) RegisterExecutor(exec Executor) {
	s.executors[exec.Name()] = exec
}

// Schedule 是调度主循环：
//  1. 获取一个本地并发令牌；
//  2. 在数据库超时内抢占一条到期任务；
//  3. 根据 Executor 字段选择执行器；
//  4. 异步执行，成功后更新 next_time；
//  5. 最终释放并发令牌，并通过 CancelFunc 停止续约、释放任务。
func (s *Scheduler) Schedule(ctx context.Context) error {
	for {
		// 放弃调度了
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := s.limiter.Acquire(ctx, 1)
		if err != nil {
			return err
		}
		dbCtx, cancel := context.WithTimeout(ctx, s.dbTimeout)
		// Preempt 成功后，Service 已经为任务启动心跳，并注入 CancelFunc。
		j, err := s.svc.Preempt(dbCtx)
		cancel()
		if err != nil {
			// 有 Error
			// 最简单的做法就是直接下一轮，也可以睡一段时间
			continue
		}

		// Executor 字段来自任务表，用于在不同执行方式之间路由。
		exec, ok := s.executors[j.Executor]
		if !ok {
			// 你可以直接中断了，也可以下一轮
			s.l.Error("找不到执行器",
				logger.Int64("jid", j.Id),
				logger.String("executor", j.Executor))
			continue
		}

		go func() {
			defer func() {
				// 无论执行成功还是失败，都要释放本地并发配额和数据库任务占用。
				s.limiter.Release(1)
				j.CancelFunc()
			}()
			err1 := exec.Exec(ctx, j)
			if err1 != nil {
				s.l.Error("执行任务失败",
					logger.Int64("jid", j.Id),
					logger.Error(err1))
				return
			}
			// 只有执行成功才推进 next_time；执行失败时释放后仍保持原到期时间，
			// 因而可以在后续调度轮次再次被抢占。
			err1 = s.svc.ResetNextTime(ctx, j)
			if err1 != nil {
				s.l.Error("重置下次执行时间失败",
					logger.Int64("jid", j.Id),
					logger.Error(err1))
			}
		}()
	}
}
