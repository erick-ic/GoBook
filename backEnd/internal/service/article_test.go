package service

import (
	"GoBook/internal/domain"
	"GoBook/internal/repository/article"
	articlerepomocks "GoBook/internal/repository/article/mocks"
	"GoBook/pkg/logger"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockgen -source=internal/repository/article/article.go -package=articlerepomocks -destination=internal/repository/article/mocks/article.mock.go
// mockgen -source=internal/repository/article/article_author.go -package=articlerepomocks -destination=internal/repository/article/mocks/article_author.mock.go
// mockgen -source=internal/repository/article/article_reader.go -package=articlerepomocks -destination=internal/repository/article/mocks/article_reader.mock.go

//go:generate mockgen -source=../repository/article/article.go -package=articlerepomocks -destination=../repository/article/mocks/article.mock.go
func Test_articleService_Publish(t *testing.T) {
	//now := time.Now()
	testCases := []struct {
		name string

		ctx     context.Context
		article domain.Article

		mock func(ctrl *gomock.Controller) (
			article.ArticleAuthorRepository,
			article.ArticleReaderRepository,
		)

		expectErr error
		expectId  int64
	}{
		{
			name: "发表成功",
			ctx:  context.Background(),
			article: domain.Article{
				Title:   "我的标题",
				Content: "我的内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			mock: func(ctrl *gomock.Controller) (
				article.ArticleAuthorRepository,
				article.ArticleReaderRepository,
			) {
				author := articlerepomocks.NewMockArticleAuthorRepository(ctrl)
				author.EXPECT().Create(gomock.Any(), domain.Article{
					Title:   "我的标题",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(1), nil)

				reader := articlerepomocks.NewMockArticleReaderRepository(ctrl)
				reader.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      1,
					Title:   "我的标题",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(1), nil)

				return author, reader
			},
			expectId:  1,
			expectErr: nil,
		},
		{
			name: "修改并发表成功",
			ctx:  context.Background(),
			article: domain.Article{
				Id:      2,
				Title:   "我的标题2",
				Content: "我的内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			mock: func(ctrl *gomock.Controller) (
				article.ArticleAuthorRepository,
				article.ArticleReaderRepository,
			) {
				author := articlerepomocks.NewMockArticleAuthorRepository(ctrl)
				author.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(nil)

				reader := articlerepomocks.NewMockArticleReaderRepository(ctrl)
				reader.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(2), nil)

				return author, reader
			},
			expectId:  2,
			expectErr: nil,
		},
		{
			name: "保存到制作库成功，线上库重试成功！",
			ctx:  context.Background(),
			article: domain.Article{
				Id:      2,
				Title:   "我的标题2",
				Content: "我的内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			mock: func(ctrl *gomock.Controller) (
				article.ArticleAuthorRepository,
				article.ArticleReaderRepository,
			) {
				author := articlerepomocks.NewMockArticleAuthorRepository(ctrl)
				author.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(nil)

				reader := articlerepomocks.NewMockArticleReaderRepository(ctrl)
				reader.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(0), errors.New("线上库保存失败！"))

				//重试
				reader.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(int64(2), nil)

				return author, reader
			},
			expectId:  2,
			expectErr: nil,
		},
		{
			name: "保存到制作库失败！",
			ctx:  context.Background(),
			article: domain.Article{
				Id:      2,
				Title:   "我的标题2",
				Content: "我的内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			mock: func(ctrl *gomock.Controller) (
				article.ArticleAuthorRepository,
				article.ArticleReaderRepository,
			) {
				author := articlerepomocks.NewMockArticleAuthorRepository(ctrl)
				author.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(errors.New("保存到制作库失败！"))

				reader := articlerepomocks.NewMockArticleReaderRepository(ctrl)
				return author, reader
			},
			expectId:  0,
			expectErr: errors.New("保存到制作库失败！"),
		},
		{
			name: "保存到制作库成功，线上库重试失败！",
			ctx:  context.Background(),
			article: domain.Article{
				Id:      2,
				Title:   "我的标题2",
				Content: "我的内容",
				Author: domain.Author{
					Id: 123,
				},
			},
			mock: func(ctrl *gomock.Controller) (
				article.ArticleAuthorRepository,
				article.ArticleReaderRepository,
			) {
				author := articlerepomocks.NewMockArticleAuthorRepository(ctrl)
				author.EXPECT().Update(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Return(nil)

				reader := articlerepomocks.NewMockArticleReaderRepository(ctrl)
				reader.EXPECT().Save(gomock.Any(), domain.Article{
					Id:      2,
					Title:   "我的标题2",
					Content: "我的内容",
					Author: domain.Author{
						Id: 123,
					},
				}).Times(3).Return(int64(0), errors.New("线上库保存失败！"))
				return author, reader
			},
			expectId:  0,
			expectErr: errors.New("线上库保存失败！"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			//初始化控制器
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			/*
				func NewArticleService(
					repo article.ArticleRepository,
					reader article.ArticleReaderRepository,
				) ArticleService
			*/
			author, reader := tc.mock(ctrl)
			svc := NewArticleServiceV1(author, reader, &logger.NopLogger{})
			id, err := svc.PublishV1(tc.ctx, tc.article)
			assert.Equal(t, tc.expectErr, err)
			assert.Equal(t, tc.expectId, id)
		})
	}
}
