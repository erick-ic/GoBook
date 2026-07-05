package memory

import (
	"context"
	"fmt"
)

type Service struct{}

func NewMemoService() *Service {
	return &Service{}
}

func (s Service) Send(ctx context.Context, tplId string, args []string, numbers ...string) error {
	fmt.Println("args:", args)
	return nil
}
