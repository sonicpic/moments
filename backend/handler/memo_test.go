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
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestInteractionDefaults(t *testing.T) {
	tests := []struct {
		name                                                             string
		content                                                          string
		legacyVisible, enableLike, showLike, enableComment, showComments bool
		want                                                             [4]bool
	}{
		{
			name:          "old configuration keeps interactions visible",
			content:       `{"enableComment":true}`,
			enableComment: true,
			want:          [4]bool{true, true, true, true},
		},
		{
			name:          "old hidden interaction switch hides visitor counts and comments",
			content:       `{"showVisitorInteractions":false,"enableComment":true}`,
			enableComment: true,
			want:          [4]bool{false, false, true, false},
		},
		{
			name:          "disabling likes always disables visible like count",
			content:       `{"enableLike":false,"showVisitorLikeCount":true,"enableComment":true,"showVisitorComments":true}`,
			enableComment: true,
			want:          [4]bool{false, false, true, true},
		},
		{
			name:    "disabling comments always disables visible comments",
			content: `{"enableLike":true,"showVisitorLikeCount":true,"enableComment":false,"showVisitorComments":true}`,
			want:    [4]bool{true, true, false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enableLike, showLike, enableComment, showComments := interactionDefaults(
				tt.content,
				tt.legacyVisible,
				tt.enableLike,
				tt.showLike,
				tt.enableComment,
				tt.showComments,
			)
			got := [4]bool{enableLike, showLike, enableComment, showComments}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
