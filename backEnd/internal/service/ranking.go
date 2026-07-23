package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository"
	"context"
	"math"
	"time"

	"github.com/ecodeclub/ekit/queue"
	"github.com/ecodeclub/ekit/slice"
)

// RankingService 排行榜服务接口
type RankingService interface {
	TopN(ctx context.Context) error
}

// BatchRankingService 批量排行榜计算服务
// 采用"分批取数 + 优先队列维护TopN"的流式算法，避免一次性加载全部文章到内存
type BatchRankingService struct {
	articleSvc ArticleService
	interSvc   InteractiveService
	batchSize  int // 每批取文章数
	topNum     int // 最终保留的 TopN 数量
	// scoreFunc 计算单篇文章得分，参数为发布时间和点赞数
	// 返回值不能为负数（优先队列要求）
	scoreFunc func(t time.Time, likeCnt int64) float64
	repo      repository.RankingRepository
}

// NewBatchRankingService 创建排行榜服务
// 默认每批读取 100 篇并保留 Top100；得分同时考虑点赞数与文章发布时间，
// 发布时间越久，时间衰减越明显。
func NewBatchRankingService(
	articleSvc ArticleService,
	interSvc InteractiveService,
	repo repository.RankingRepository,
) RankingService {
	return &BatchRankingService{
		articleSvc: articleSvc,
		interSvc:   interSvc,
		batchSize:  100,
		topNum:     100,
		scoreFunc: func(t time.Time, likeCnt int64) float64 {
			duration := time.Since(t).Seconds()
			return float64(likeCnt-1) / math.Pow(duration+2, 1.5)
		},
		repo: repo,
	}
}

// TopN 是定时任务入口：完成排行榜计算后输出本次结果。
// 后续若要对外提供榜单，可在这里把结果交给 RankingRepository 持久化。
func (br *BatchRankingService) TopN(ctx context.Context) error {
	articles, err := br.topN(ctx)
	if err != nil {
		return err
	}
	//放入redis缓存
	err = br.repo.ReplaceTopN(ctx, articles)
	if err != nil {
		return err
	}
	return nil
}

// topN 核心算法：分批取文章，维护大小为 topNum 的优先队列
//
// 算法流程：
//  1. 分批从数据库取已发布文章（按时间倒序）
//  2. 批量查询对应互动数据（点赞数）
//  3. 计算每篇文章得分，维护最小堆（优先队列）
//     - 队列未满：直接入队
//     - 队列已满：与队首（最小值）比较，保留高分文章
//  4. 终止条件：取不够一批 或 文章已超7天
//  5. 从队列取出结果，按得分降序排列
func (br *BatchRankingService) topN(ctx context.Context) ([]domain.Article, error) {
	offset := 0
	start := time.Now()
	// ddl：7天前的截止时间，只取7天内发布的文章（业务需求）
	ddl := start.Add(-7 * 24 * time.Hour)

	type Score struct {
		art   domain.Article
		score float64
	}

	// 优先队列：小顶堆，队首为当前 TopN 中得分最低的文章
	// 比较函数：src.score > dst.score 返回 1 表示 src 在 dst 之后（小顶堆）
	topN := queue.NewConcurrentPriorityQueue[Score](
		br.topNum,
		func(src Score, dst Score) int {
			if src.score > dst.score {
				return 1
			} else if src.score == dst.score {
				return 0
			}

			return -1
		},
	)

	for {
		// 1. 取一批已发布文章
		arts, err := br.articleSvc.ListPublishedArticles(ctx, start, offset, br.batchSize)
		if err != nil {
			return nil, err
		}

		// 提取文章ID，用于批量查询互动数据
		ids := slice.Map[domain.Article, int64](arts, func(idx int, src domain.Article) int64 {
			return src.Id
		})

		// 2. 批量查询对应互动数据（点赞数等）
		inters, err := br.interSvc.GetByIds(ctx, "article", ids)
		if err != nil {
			return nil, err
		}

		// 3. 合并文章和互动数据，计算分数，维护TopN队列
		for _, art := range arts {
			inter, ok := inters[art.Id]
			if !ok {
				// 没有互动数据，跳过（不计入排行榜）
				continue
			}
			score := br.scoreFunc(art.Utime, inter.LikeCnt)

			ele := Score{
				art:   art,
				score: score,
			}
			err = topN.Enqueue(ele)
			if err == queue.ErrOutOfCapacity {
				// 队列已满，与当前最小值比较，只保留高分文章
				minEle, _ := topN.Dequeue()
				if minEle.score < score {
					// 新文章得分更高，替换最小值
					_ = topN.Enqueue(ele)
				} else {
					// 新文章得分更低或相等，保留原最小值
					_ = topN.Enqueue(minEle)
				}
			}
		}

		// 更新offset，准备取下一批
		offset += len(arts)

		// 终止条件：取不够一批（说明后面没有更多文章了）
		// 或 本批最后一条文章已超7天（时间优化，后续文章更旧）
		if len(arts) < br.batchSize ||
			(len(arts) > 0 && arts[len(arts)-1].Utime.Before(ddl)) {
			break
		}
	}

	// 4. 从优先队列取出结果（队首是最小值，所以从尾部填数组实现降序）
	res := make([]domain.Article, topN.Len())
	for i := topN.Len() - 1; i >= 0; i-- {
		ele, _ := topN.Dequeue()
		res[i] = ele.art
	}
	return res, nil
}
