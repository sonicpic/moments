package handler

import (
	"testing"
	"time"

	"github.com/kingwrcy/moments/vo"
)

func TestExternalAccessStartAt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  *time.Time
	}{
		{
			name:  "empty value is unrestricted",
			value: " ",
		},
		{
			name:  "HTML datetime local value",
			value: "2026-07-22T08:30",
			want:  timePtr(time.Date(2026, time.July, 22, 8, 30, 0, 0, time.Local)),
		},
		{
			name:  "date-only value",
			value: "2026-07-22",
			want:  timePtr(time.Date(2026, time.July, 22, 0, 0, 0, 0, time.Local)),
		},
		{
			name:  "invalid value is unrestricted",
			value: "not-a-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := externalAccessStartAt(vo.FullSysConfigVO{ExternalAccessStartAt: tt.value})
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %s", got)
				}
				return
			}
			if got == nil || !got.Equal(*tt.want) {
				t.Fatalf("expected %s, got %v", tt.want, got)
			}
		})
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
