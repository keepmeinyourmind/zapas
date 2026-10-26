package profile

import (
	"testing"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func TestTrust(t *testing.T) {
	tests := []struct {
		name   string
		months int
		want   domain.TrustLevel
	}{
		{"нет истории — низкое доверие", 0, domain.TrustLevelLow},
		{"2 месяца — низкое", 2, domain.TrustLevelLow},
		{"3 месяца — среднее", 3, domain.TrustLevelMid},
		{"5 месяцев — среднее", 5, domain.TrustLevelMid},
		{"6 месяцев — высокое", 6, domain.TrustLevelHigh},
		{"год — высокое", 12, domain.TrustLevelHigh},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Trust(tt.months); got != tt.want {
				t.Errorf("Trust(%d) = %d, want %d", tt.months, got, tt.want)
			}
		})
	}
}
