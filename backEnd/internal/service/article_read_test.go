package service

import (
	"GoBook/internal/domain"
	events "GoBook/internal/events/article"
	articlerepomocks "GoBook/internal/repository/article/mocks"
	"GoBook/pkg/logger"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestArticleService_GetByPubIdProducesReadEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := articlerepomocks.NewMockArticleRepository(ctrl)
	producer := &readEventProducer{events: make(chan events.ReadEvent, 1)}
	svc := NewArticleService(repo, producer, &logger.NopLogger{})
	wantArticle := domain.Article{Id: 1, Title: "标题"}

	repo.EXPECT().
		GetByPubId(gomock.Any(), int64(1)).
		Return(wantArticle, nil)

	got, err := svc.GetByPubId(context.Background(), 1, 2)

	require.NoError(t, err)
	assert.Equal(t, wantArticle, got)
	select {
	case event := <-producer.events:
		assert.Equal(t, events.ReadEvent{
			Uid:       2,
			ArticleId: 1,
		}, event)
	case <-time.After(time.Second):
		t.Fatal("等待阅读事件超时")
	}
}

type readEventProducer struct {
	events chan events.ReadEvent
}

func (p *readEventProducer) ProduceReadEvent(event events.ReadEvent) error {
	p.events <- event
	return nil
}
