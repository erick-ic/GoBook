package auth

import (
	"GoBook/internal/service/sms"
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

// Service 是短信发送服务的装饰器，它在底层 sms.Service 之上增加了 JWT 鉴权能力。
// 调用方必须传入一个合法的 JWT 作为业务凭证（biz 参数），
// 该 JWT 中携带了业务方允许使用的短信模板 ID（Tpl）。
type Service struct {
	svc sms.Service // 被装饰的底层短信发送服务
	key string      // 对称密钥，用于验证 JWT 签名（由服务端线下统一管理）
}

// Send 发送短信。
// 参数 biz 不是普通的业务标识，而是一个 JWT 字符串（由业务方线下申请获得）。
// 该 JWT 的载荷（Payload）中必须包含标准字段（如 exp）和自定义字段 Tpl（模板ID）。
// 方法会解析并验证 JWT，若通过则提取 Tpl 作为模板 ID 调用底层服务。
func (s *Service) Send(ctx context.Context, biz string, args []string, numbers ...string) error {
	// ---------- 1. 准备接收 JWT 载荷的结构体 ----------
	// tc 用于存储解析后的 Claims（声明）。
	// 注意：Claims 嵌入了 jwt.RegisteredClaims，因此能自动处理 exp、iat 等标准字段。
	var tc Claims

	// ---------- 2. 解析并验证 JWT ----------
	// jwt.ParseWithClaims 的三个参数：
	//   - biz：JWT 字符串（由调用方通过参数传入，即“token 的存储与传递方式”）
	//   - &tc：解析成功后，载荷数据会填充到 tc 中
	//   - 回调函数：提供密钥用于验证签名（固定返回 s.key）
	//
	// 解码过程详解：
	//   a) 库内部将 JWT 拆分为 Header、Payload、Signature 三部分
	//   b) 用回调函数提供的密钥重新计算签名，与传入的 Signature 比对
	//   c) 若签名一致，再检查 Payload 中的 exp 等标准声明是否有效
	//   d) 全部通过后，将 Payload 的 JSON 反序列化到 tc 中
	token, err := jwt.ParseWithClaims(biz, &tc, func(*jwt.Token) (interface{}, error) {
		// 这里固定使用同一个密钥进行对称加密验证（HMAC 系列算法）。
		// 如果有多个密钥（多租户），可在此根据 token.Header["kid"] 动态选择。
		return s.key, nil
	})
	if err != nil {
		// 可能原因：签名错误、过期、格式非法、密钥不匹配等
		return err
	}

	// ---------- 3. 二次确认 token 有效（防御性编程） ----------
	if !token.Valid {
		return errors.New("token is invalid")
	}

	// ---------- 4. 从载荷中提取业务参数并调用底层服务 ----------
	// 此时 tc.Tpl 已经安全地从 JWT 中取出，且经过了签名验证，确保未被篡改。
	// 将 Tpl 作为模板 ID 传给真正的短信发送逻辑。
	return s.svc.Send(ctx, tc.Tpl, args, numbers...)
}

// Claims 自定义声明结构体，定义了 JWT 载荷中允许包含的字段。
// 嵌入 jwt.RegisteredClaims 以支持标准字段（exp, iat, iss, sub 等）。
type Claims struct {
	jwt.RegisteredClaims        // 标准声明（过期时间、签发时间等）
	Tpl                  string `json:"Tpl"` // 自定义业务字段：短信模板 ID
}
