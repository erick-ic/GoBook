package client

import (
	interactivev1 "GoBook/api/proto/gen/interactive/v1"
	"context"
	"fmt"
	"math/rand"

	"github.com/ecodeclub/ekit/syncx/atomicx"
	"google.golang.org/grpc"
)

// GreyScaleInteractiveServiceClient 在远程 gRPC 客户端和本地适配器之间按比例分流。
// 每次 RPC 都会独立选择调用目标，因此它适合迁移期流量验证，但不提供用户级粘性。
type GreyScaleInteractiveServiceClient struct {
	remote    interactivev1.InteractiveServiceClient
	local     interactivev1.InteractiveServiceClient
	threshold *atomicx.Value[int32]
}

var _ interactivev1.InteractiveServiceClient = (*GreyScaleInteractiveServiceClient)(nil)

// NewGreyScaleInteractiveServiceClient 创建支持按比例灰度的互动服务客户端。
// threshold 的有效范围是 [0, 100]：0 表示全部走本地，100 表示全部走远程。
func NewGreyScaleInteractiveServiceClient(
	remote interactivev1.InteractiveServiceClient,
	local interactivev1.InteractiveServiceClient,
	threshold int32,
) (*GreyScaleInteractiveServiceClient, error) {
	if err := validateThreshold(threshold); err != nil {
		return nil, err
	}
	return &GreyScaleInteractiveServiceClient{
		remote:    remote,
		local:     local,
		threshold: atomicx.NewValueOf(threshold),
	}, nil
}

// IncrReadCnt 按当前灰度阈值转发阅读数递增请求。
func (g *GreyScaleInteractiveServiceClient) IncrReadCnt(ctx context.Context, in *interactivev1.IncrReadCntRequest, opts ...grpc.CallOption) (*interactivev1.IncrReadCntResponse, error) {
	return g.client().IncrReadCnt(ctx, in, opts...)
}

// Like 按当前灰度阈值转发点赞请求。
func (g *GreyScaleInteractiveServiceClient) Like(ctx context.Context, in *interactivev1.LikeRequest, opts ...grpc.CallOption) (*interactivev1.LikeResponse, error) {
	return g.client().Like(ctx, in, opts...)
}

// CancelLike 按当前灰度阈值转发取消点赞请求。
func (g *GreyScaleInteractiveServiceClient) CancelLike(ctx context.Context, in *interactivev1.CancelLikeRequest, opts ...grpc.CallOption) (*interactivev1.CancelLikeResponse, error) {
	return g.client().CancelLike(ctx, in, opts...)
}

// Get 按当前灰度阈值查询单个业务对象的互动数据。
func (g *GreyScaleInteractiveServiceClient) Get(ctx context.Context, in *interactivev1.GetRequest, opts ...grpc.CallOption) (*interactivev1.GetResponse, error) {
	return g.client().Get(ctx, in, opts...)
}

// GetByIds 按当前灰度阈值批量查询互动数据。
func (g *GreyScaleInteractiveServiceClient) GetByIds(ctx context.Context, in *interactivev1.GetByIdsRequest, opts ...grpc.CallOption) (*interactivev1.GetByIdsResponse, error) {
	return g.client().GetByIds(ctx, in, opts...)
}

// Collect 按当前灰度阈值转发收藏请求。
func (g *GreyScaleInteractiveServiceClient) Collect(ctx context.Context, in *interactivev1.CollectRequest, opts ...grpc.CallOption) (*interactivev1.CollectResponse, error) {
	return g.client().Collect(ctx, in, opts...)
}

// UpdateThreshold 原子更新灰度阈值；非法配置不会覆盖当前有效值。
func (g *GreyScaleInteractiveServiceClient) UpdateThreshold(newThreshold int32) error {
	if err := validateThreshold(newThreshold); err != nil {
		return err
	}
	g.threshold.Store(newThreshold)
	return nil
}

// client 按远程流量百分比选择本次 RPC 的实际执行方。
func (g *GreyScaleInteractiveServiceClient) client() interactivev1.InteractiveServiceClient {
	threshold := g.threshold.Load()
	// 生成 [0, 100) 的随机数，小于阈值时走远程调用。
	num := rand.Int31n(100)

	if num < threshold {
		return g.remote
	}
	return g.local
}

func validateThreshold(threshold int32) error {
	if threshold < 0 || threshold > 100 {
		return fmt.Errorf("互动服务灰度阈值必须在 0 到 100 之间，当前值为 %d", threshold)
	}
	return nil
}
