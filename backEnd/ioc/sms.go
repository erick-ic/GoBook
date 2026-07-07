package ioc

import (
	"GoBook/internal/service/sms"
	"GoBook/internal/service/sms/memory"
)

func InitSMSService() sms.Service {
	//方便更换SMS服务
	return memory.NewMemoService()
}
