package job

import (
	"testing"

	"go.uber.org/mock/gomock"
)

func TestRankingJob(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

		})
	}
}
