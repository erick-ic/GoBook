package service

import (
	"GoBook/internal/domain"
	svcmocks "GoBook/internal/service/mocks"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestRankingTopN 测试排行榜 TopN 计算
// 验证"分批取数 + 优先队列"的流式算法是否正确：
//  1. 模拟3批文章数据（每批2条，最后一批为空）
//  2. 模拟对应的互动数据（点赞数）
//  3. 使用简单 scoreFunc（score=likeCnt）验证排序结果
//  4. 期望结果：按点赞数降序取 Top3 → [Id4(400), Id3(300), Id2(200)]
func TestRankingTopN(t *testing.T) {
	const batchSize = 2
	now := time.Now()

	testCases := []struct {
		name           string
		mock           func(ctrl *gomock.Controller) (ArticleService, InteractiveService)
		expectErr      error
		expectArticles []domain.Article
	}{
		{
			name: "计算成功",
			mock: func(ctrl *gomock.Controller) (ArticleService, InteractiveService) {
				artSvc := svcmocks.NewMockArticleService(ctrl)

				// 模拟第1批：offset=0, limit=2 → [Id:1, Id:2]
				artSvc.EXPECT().ListPublishedArticles(gomock.Any(), gomock.Any(), 0, batchSize).
					Return([]domain.Article{
						{Id: 1, Utime: now, Ctime: now},
						{Id: 2, Utime: now, Ctime: now},
					}, nil)
				// 模拟第2批：offset=2, limit=2 → [Id:3, Id:4]
				artSvc.EXPECT().ListPublishedArticles(gomock.Any(), gomock.Any(), 2, batchSize).
					Return([]domain.Article{
						{Id: 3, Utime: now, Ctime: now},
						{Id: 4, Utime: now, Ctime: now},
					}, nil)
				// 模拟第3批：offset=4, limit=2 → 空（没有更多数据）
				artSvc.EXPECT().ListPublishedArticles(gomock.Any(), gomock.Any(), 4, batchSize).
					Return([]domain.Article{}, nil)

				interSvc := svcmocks.NewMockInteractiveService(ctrl)
				// 第1批互动数据：Id1(100赞), Id2(200赞)
				interSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{1, 2}).
					Return(map[int64]domain.Interactive{
						1: {BizId: 1, LikeCnt: 100},
						2: {BizId: 2, LikeCnt: 200},
					}, nil)
				// 第2批互动数据：Id3(300赞), Id4(400赞)
				interSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{3, 4}).
					Return(map[int64]domain.Interactive{
						3: {BizId: 3, LikeCnt: 300},
						4: {BizId: 4, LikeCnt: 400},
					}, nil)
				// 第3批互动数据：空ID列表，返回空map
				interSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{}).
					Return(map[int64]domain.Interactive{}, nil)

				return artSvc, interSvc
			},
			expectErr: nil,
			// 期望 Top3 降序：Id4(400) > Id3(300) > Id2(200) > Id1(100)
			expectArticles: []domain.Article{
				{Id: 4, Utime: now, Ctime: now},
				{Id: 3, Utime: now, Ctime: now},
				{Id: 2, Utime: now, Ctime: now},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建 gomock 控制器，用于管理 mock 对象的预期调用
			ctrl := gomock.NewController(t)
			defer ctrl.Finish() // 测试结束时验证所有 EXPECT 是否都被调用

			artSvc, interSvc := tc.mock(ctrl)
			svc := &BatchRankingService{
				articleSvc: artSvc,
				interSvc:   interSvc,
				batchSize:  batchSize,
				topNum:     3, // 取 Top3
				scoreFunc: func(t time.Time, likeCnt int64) float64 {
					// 测试用简化得分函数：得分=点赞数
					return float64(likeCnt)
				},
			}

			arts, err := svc.topN(context.Background())
			assert.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectArticles, arts)
		})
	}
}
