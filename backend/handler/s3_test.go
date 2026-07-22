package handler

import "testing"

func TestJoinResourceURL(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		objectKey string
		expected  string
	}{
		{
			name:      "domain without trailing slash",
			domain:    "https://img.example.com",
			objectKey: "2026/07/21/a.jpg",
			expected:  "https://img.example.com/2026/07/21/a.jpg",
		},
		{
			name:      "domain with trailing slash",
			domain:    "https://img.example.com/",
			objectKey: "2026/07/21/a.jpg",
			expected:  "https://img.example.com/2026/07/21/a.jpg",
		},
		{
			name:      "object key with leading slash",
			domain:    "https://img.example.com",
			objectKey: "/2026/07/21/a.jpg",
			expected:  "https://img.example.com/2026/07/21/a.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := joinResourceURL(tt.domain, tt.objectKey); actual != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, actual)
			}
		})
	}
}
