package service

import (
	"GoBook/internal/repository"
	"GoBook/internal/service/sms"
	"context"
	"fmt"
	"math/rand"
)

const codeTplId = "1877556"

var (
	ErrCodeSendTooMany        = repository.ErrCodeSendTooMany
	ErrCodeVerifyTooManyTimes = repository.ErrCodeVerifyTooManyTimes
)

type CodeService struct {
	repo *repository.CodeRepository
	sms  sms.Service
}

func NewCodeService(repo *repository.CodeRepository, sms sms.Service) *CodeService {
	return &CodeService{
		repo: repo,
		sms:  sms,
	}
}

// Send 发送验证码
func (cs *CodeService) Send(ctx context.Context, biz, phone string) error {
	//1.生成验证码
	code := cs.generateCode()
	//2.存入redis
	err := cs.repo.Store(ctx, biz, phone, code)
	if err != nil {
		return err
	}
	//3.存入redis完成后，开始发送
	err = cs.sms.Send(ctx, codeTplId, []string{code}, phone)
	return err
}

// Verify 验证码校验
func (cs *CodeService) Verify(ctx context.Context, biz, phone, code string) (bool, error) {
	return cs.repo.VerifyCode(ctx, biz, phone, code)
}

// 生成6位随机验证码
func (cs *CodeService) generateCode() string {
	//六位数，num在0， 99999之间包含0，99999
	num := rand.Intn(1000000)
	//不足六位的补零
	return fmt.Sprintf("%06d", num)
}
