package ioc

import (
	interactivegrpc "GoBook/interactive/grpc"
	"GoBook/pkg/grpcx"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// InitGRPCxServer 创建 gRPC Server，并注册互动服务实现。
func InitGRPCxServer(interServer *interactivegrpc.InteractiveServiceServer) *grpcx.Server {
	type config struct {
		Addr string `mapstructure:"addr"`
	}
	var cfg config
	if err := viper.UnmarshalKey("grpc.server", &cfg); err != nil {
		panic(err)
	}
	if cfg.Addr == "" {
		panic("interactive: grpc.server.addr 不能为空")
	}

	server := grpc.NewServer()
	interServer.RegisterServer(server)

	return &grpcx.Server{
		Server: server,
		Addr:   cfg.Addr,
	}
}
