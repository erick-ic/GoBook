package grpc

import (
	"net"
	"testing"

	"google.golang.org/grpc"
)

func TestServer(t *testing.T) {
	// 创建只负责协议传输和请求调度的 gRPC 服务实例。
	server := grpc.NewServer()

	// 测试退出时停止接收新请求，并等待正在处理的请求完成。
	defer server.GracefulStop()

	// 注册业务实现，使生成的 UserService 路由能够分发到 Server。
	userServer := &Server{}
	RegisterUserServiceServer(server, userServer)

	// 地址前的冒号表示监听本机所有网络接口的 8090 端口。
	l, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}

	// Serve 会阻塞并持续接收连接，直到服务被停止或监听器返回错误。
	er := server.Serve(l)
	t.Log(er)
}
