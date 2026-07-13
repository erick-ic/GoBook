package logger

//方式1：兼容性最好
//type Logger interface {
//	Debug(msg string, args ...any)
//	Info(msg string, args ...any)
//	Warn(msg string, args ...any)
//	Error(msg string, args ...any)
//}

//func LoggerFunc() {
//	var l Logger
//	l.Info("用户未注册，id为%s", 21)
//}

// 方式2: 参数需要命名约束
// 业务使用的日志接口，所有日志方法接收结构化字段（Field 切片）。
type LoggerV1 interface {
	Debug(msg string, args ...Field)
	Info(msg string, args ...Field)
	Warn(msg string, args ...Field)
	Error(msg string, args ...Field)
}

type Field struct {
	Key   string
	Value any
}

func LoggerV1Func() {
	var l LoggerV1
	l.Info("用户未注册，id为", Field{
		Key:   "id",
		Value: 21,
	})
}

//// 方式3:需要完善代码评审流程，介于方式1和方式2之间
//type LoggerV2 interface {
//	//args必须是偶数，并且按照key-value，key-value
//	Debug(msg string, args ...any)
//	Info(msg string, args ...any)
//	Warn(msg string, args ...any)
//	Error(msg string, args ...any)
//}
//
//func LoggerV2Func() {
//	var l LoggerV2
//	l.Info("用户未注册，id为", "id", 21)
//	//错误使用
//	//l.Info("用户未注册，id为", "id", 21, "ccc")
//}
