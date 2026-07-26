package tencent

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	sms "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
)

// TestSender 会调用腾讯云并真实发送短信，因此默认跳过。
// 手动验证时需显式设置 RUN_TENCENT_SMS_TEST=1、腾讯云密钥和接收手机号。
func TestSender(t *testing.T) {
	if os.Getenv("RUN_TENCENT_SMS_TEST") != "1" {
		t.Skip("手动短信集成测试未开启")
	}
	secretId, secretIDOK := os.LookupEnv("SMS_SECRET_ID")
	secretKey, secretKeyOK := os.LookupEnv("SMS_SECRET_KEY")
	phoneNumber, phoneOK := os.LookupEnv("SMS_PHONE_NUMBER")
	if !secretIDOK || !secretKeyOK || !phoneOK {
		t.Fatal("请设置 SMS_SECRET_ID、SMS_SECRET_KEY 和 SMS_PHONE_NUMBER")
	}

	c, err := sms.NewClient(common.NewCredential(secretId, secretKey),
		"ap-nanjing",
		profile.NewClientProfile())
	if err != nil {
		t.Fatal(err)
	}

	s := NewService(c, "1400842696", "妙影科技")

	testCases := []struct {
		name    string
		tplId   string
		params  []string
		numbers []string
		wantErr error
	}{
		{
			name:    "发送验证码",
			tplId:   "1877556",
			params:  []string{"123456"},
			numbers: []string{phoneNumber},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			er := s.Send(context.Background(), tc.tplId, tc.params, tc.numbers...)
			assert.Equal(t, tc.wantErr, er)
		})
	}
}
