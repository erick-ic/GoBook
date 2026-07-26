package grpc

import (
	interactivev1 "GoBook/api/proto/gen/interactive/v1"
	"GoBook/interactive/domain"
	"GoBook/interactive/service"
	"context"

	"google.golang.org/grpc"
)

// InteractiveServiceServer 将内部互动服务适配为 protobuf 定义的 gRPC 接口。
type InteractiveServiceServer struct {
	interactivev1.UnimplementedInteractiveServiceServer
	svc service.InteractiveService
}

// NewInteractiveServiceServer 创建互动 gRPC 服务端。
func NewInteractiveServiceServer(svc service.InteractiveService) *InteractiveServiceServer {
	return &InteractiveServiceServer{svc: svc}
}

// RegisterServer 将互动服务注册到给定的 gRPC Server。
func (i *InteractiveServiceServer) RegisterServer(server *grpc.Server) {
	interactivev1.RegisterInteractiveServiceServer(server, i)
}

// IncrReadCnt 增加业务对象的阅读数。
func (i *InteractiveServiceServer) IncrReadCnt(ctx context.Context, request *interactivev1.IncrReadCntRequest) (*interactivev1.IncrReadCntResponse, error) {
	err := i.svc.IncrReadCnt(ctx, request.GetBiz(), request.GetBizId())
	if err != nil {
		return nil, err
	}
	return &interactivev1.IncrReadCntResponse{}, nil
}

// Like 记录当前用户对业务对象的点赞。
func (i *InteractiveServiceServer) Like(ctx context.Context, request *interactivev1.LikeRequest) (*interactivev1.LikeResponse, error) {
	err := i.svc.Like(ctx, request.GetBiz(), request.GetBizId(), request.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.LikeResponse{}, nil
}

// CancelLike 取消当前用户对业务对象的点赞。
func (i *InteractiveServiceServer) CancelLike(ctx context.Context, request *interactivev1.CancelLikeRequest) (*interactivev1.CancelLikeResponse, error) {
	err := i.svc.CancelLike(ctx, request.GetBiz(), request.GetBizId(), request.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.CancelLikeResponse{}, nil
}

// Get 返回聚合互动数据及当前用户的互动状态。
func (i *InteractiveServiceServer) Get(ctx context.Context, request *interactivev1.GetRequest) (*interactivev1.GetResponse, error) {
	res, err := i.svc.Get(ctx, request.GetBiz(), request.GetBizId(), request.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.GetResponse{
		Inter: i.toDTO(res),
	}, nil
}

// GetByIds 批量返回互动数据，结果以业务对象 ID 为键。
func (i *InteractiveServiceServer) GetByIds(ctx context.Context, request *interactivev1.GetByIdsRequest) (*interactivev1.GetByIdsResponse, error) {
	res, err := i.svc.GetByIds(ctx, request.GetBiz(), request.GetIds())
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

// Collect 将业务对象加入用户指定的收藏夹。
func (i *InteractiveServiceServer) Collect(ctx context.Context, request *interactivev1.CollectRequest) (*interactivev1.CollectResponse, error) {
	err := i.svc.Collect(ctx, request.GetBiz(), request.GetBizId(), request.GetCid(), request.GetUid())
	if err != nil {
		return nil, err
	}
	return &interactivev1.CollectResponse{}, nil
}

// toDTO 将领域模型转换成 protobuf DTO。
func (i *InteractiveServiceServer) toDTO(inter domain.Interactive) *interactivev1.Interactive {
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
