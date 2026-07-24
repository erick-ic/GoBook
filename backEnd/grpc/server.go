package grpc

import "context"

// Server 是 UserService 的具体实现。
// 嵌入生成的 UnimplementedUserServiceServer 可在协议新增方法时保持向前兼容。
type Server struct {
	UnimplementedUserServiceServer
}

// GetById 处理按 ID 查询用户的 RPC。
// 当前是用于验证 gRPC 调用链的最小示例，因此暂未读取请求参数或访问持久化存储。
func (s *Server) GetById(ctx context.Context, request *GetByIdRequest) (*GetByIdResponse, error) {
	return &GetByIdResponse{
		User: &User{
			Id:   21,
			Name: "Tom",
		},
	}, nil
}
