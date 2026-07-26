package client

import (
	interactivev1 "GoBook/api/proto/gen/interactive/v1"
	"GoBook/interactive/domain"
	"GoBook/interactive/service"
	"context"

	"google.golang.org/grpc"
)

// InteractiveServiceAdapter 将进程内互动服务适配成 protobuf 客户端接口。
// 灰度客户端借此用统一方式调用本地和远程实现；grpc.CallOption 在本地路径中不会生效。
type InteractiveServiceAdapter struct {
	svc service.InteractiveService
}

var _ interactivev1.InteractiveServiceClient = (*InteractiveServiceAdapter)(nil)

// NewInteractiveServiceAdapter 创建进程内互动服务的客户端适配器。
func NewInteractiveServiceAdapter(svc service.InteractiveService) *InteractiveServiceAdapter {
	return &InteractiveServiceAdapter{svc: svc}
}

// IncrReadCnt 将 protobuf 请求转换为本地服务调用。
func (i *InteractiveServiceAdapter) IncrReadCnt(ctx context.Context, in *interactivev1.IncrReadCntRequest, opts ...grpc.CallOption) (*interactivev1.IncrReadCntResponse, error) {
	err := i.svc.IncrReadCnt(ctx, in.GetBiz(), in.GetBizId())
	return &interactivev1.IncrReadCntResponse{}, err
}

// Like 将 protobuf 点赞请求转发给本地服务。
func (i *InteractiveServiceAdapter) Like(ctx context.Context, in *interactivev1.LikeRequest, opts ...grpc.CallOption) (*interactivev1.LikeResponse, error) {
	err := i.svc.Like(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	return &interactivev1.LikeResponse{}, err
}

// CancelLike 将 protobuf 取消点赞请求转发给本地服务。
func (i *InteractiveServiceAdapter) CancelLike(ctx context.Context, in *interactivev1.CancelLikeRequest, opts ...grpc.CallOption) (*interactivev1.CancelLikeResponse, error) {
	err := i.svc.CancelLike(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	return &interactivev1.CancelLikeResponse{}, err
}

// Get 调用本地服务，并将领域模型转换为 protobuf 响应。
func (i *InteractiveServiceAdapter) Get(ctx context.Context, in *interactivev1.GetRequest, opts ...grpc.CallOption) (*interactivev1.GetResponse, error) {
	res, err := i.svc.Get(ctx, in.GetBiz(), in.GetBizId(), in.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.GetResponse{
		Inter: i.toDTO(res),
	}, nil
}

// GetByIds 批量调用本地服务，并按业务对象 ID 构造 protobuf Map。
func (i *InteractiveServiceAdapter) GetByIds(ctx context.Context, in *interactivev1.GetByIdsRequest, opts ...grpc.CallOption) (*interactivev1.GetByIdsResponse, error) {
	res, err := i.svc.GetByIds(ctx, in.GetBiz(), in.GetIds())
	if err != nil {
		return nil, err
	}
	m := make(map[int64]*interactivev1.Interactive, len(res))
	for k, v := range res {
		m[k] = i.toDTO(v)
	}
	return &interactivev1.GetByIdsResponse{
		Inters: m,
	}, nil
}

// Collect 将 protobuf 收藏请求转发给本地服务。
func (i *InteractiveServiceAdapter) Collect(ctx context.Context, in *interactivev1.CollectRequest, opts ...grpc.CallOption) (*interactivev1.CollectResponse, error) {
	err := i.svc.Collect(ctx, in.GetBiz(), in.GetBizId(), in.GetCid(), in.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.CollectResponse{}, nil
}

// toDTO 将本地领域模型转换为跨进程传输使用的 protobuf DTO。
func (i *InteractiveServiceAdapter) toDTO(inter domain.Interactive) *interactivev1.Interactive {
	return &interactivev1.Interactive{
		BizId:      inter.BizId,
		Biz:        inter.Biz,
		ReadCnt:    inter.ReadCnt,
		LikeCnt:    inter.LikeCnt,
		CollectCnt: inter.CollectCnt,
		Liked:      inter.Liked,
		Collected:  inter.Collected,
	}
}
