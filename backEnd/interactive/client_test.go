//go:build integration

package main

import (
	interactivev1 "GoBook/api/proto/gen/interactive/v1"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestClient 是需要互动服务已监听 :8090 的手工联调测试。
// 使用 -tags=integration 显式运行，避免普通 go test 依赖外部进程。
func TestClient(t *testing.T) {
	cc, err := grpc.NewClient("localhost:8090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer cc.Close()

	client := interactivev1.NewInteractiveServiceClient(cc)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.Get(ctx, &interactivev1.GetRequest{
		Biz:   "test",
		BizId: 3,
		Uid:   999,
	})
	require.NoError(t, err)
	t.Log(resp.Inter)
}
