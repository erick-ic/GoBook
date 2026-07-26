package grpcx

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

// Server 为 gRPC Server 补充可配置的监听地址和统一启动入口。
type Server struct {
	*grpc.Server
	Addr string
}

// Serve 在 Addr 上监听并阻塞处理 gRPC 请求。
func (s *Server) Serve() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("grpcx: 监听 %s 失败: %w", s.Addr, err)
	}

	return s.Server.Serve(l)
}
