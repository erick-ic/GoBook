package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClient(t *testing.T) {
	// 客户端测试依赖已有的 gRPC 服务监听 8090 端口；本测试本身不会启动服务端。
	// Dial 默认异步建立连接，因此服务不可用等错误通常会在首次 RPC 时返回。
	cc, err := grpc.Dial(":8090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	// NewUserServiceClient 使用 user.proto 生成的客户端桩，将方法调用转换为 gRPC 请求。
	client := NewUserServiceClient(cc)

	// 为 RPC 设置超时时间，避免服务端无响应时测试永久阻塞。
	ctx, cancle := context.WithTimeout(context.Background(), time.Minute)
	defer cancle()

	// 调用远端 UserService.GetById；网络和服务端错误会通过 er 返回。
	resp, er := client.GetById(ctx, &GetByIdRequest{
		Id: 21,
	})

	assert.NoError(t, er)

	// RPC 成功时记录反序列化后的用户信息，便于观察响应内容。
	t.Log(resp.User)
}
