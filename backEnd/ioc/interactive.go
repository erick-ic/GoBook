package ioc

import (
	interactivev1 "GoBook/api/proto/gen/interactive/v1"
	"GoBook/interactive/service"
	"GoBook/internal/web/client"
	"log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InitInteractiveGRPCClient 创建互动服务的本地适配器、远程客户端和灰度路由器。
// 配置文件热更新时只调整流量阈值；地址和安全配置变更需要重启应用以重新建连。
func InitInteractiveGRPCClient(svc service.InteractiveService) interactivev1.InteractiveServiceClient {
	type Config struct {
		Addr      string `mapstructure:"addr"`
		Secure    bool   `mapstructure:"secure"`
		Threshold int32  `mapstructure:"threshold"`
	}
	var config Config
	err := viper.UnmarshalKey("grpc.client.interactive", &config)
	if err != nil {
		panic(err)
	}

	var opts []grpc.DialOption
	if config.Secure {
		// 生产环境应在此加载 TLS 证书；当前配置尚未实现安全连接。
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	cc, err := grpc.Dial(config.Addr, opts...)
	if err != nil {
		panic(err)
	}
	// 远程客户端执行真实 RPC，本地适配器用于灰度期间保留单体实现。
	remote := interactivev1.NewInteractiveServiceClient(cc)
	local := client.NewInteractiveServiceAdapter(svc)
	res, err := client.NewGreyScaleInteractiveServiceClient(remote, local, config.Threshold)
	if err != nil {
		panic(err)
	}

	// WatchConfig 由主程序统一启动，这里只订阅阈值变化并原子更新路由比例。
	viper.OnConfigChange(func(fsnotify.Event) {
		cfg := Config{}
		if err := viper.UnmarshalKey("grpc.client.interactive", &cfg); err != nil {
			log.Printf("重新加载互动服务灰度配置失败: %v", err)
			return
		}
		if err := res.UpdateThreshold(cfg.Threshold); err != nil {
			log.Printf("忽略非法的互动服务灰度配置: %v", err)
		}
	})
	return res
}
