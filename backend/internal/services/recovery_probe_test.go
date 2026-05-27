package services

import "testing"

func TestJoinUpstreamURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name:    "base url without trailing slash",
			baseURL: "https://api.example.com",
			path:    "/v1/messages",
			want:    "https://api.example.com/v1/messages",
		},
		{
			name:    "base url with trailing slash",
			baseURL: "https://api.example.com/",
			path:    "/v1/messages",
			want:    "https://api.example.com/v1/messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinUpstreamURL(tt.baseURL, tt.path); got != tt.want {
				t.Fatalf("joinUpstreamURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChooseProbeModel(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		want string
	}{
		{
			name: "prefers haiku",
			ids:  []string{"claude-sonnet-4", "claude-haiku-3"},
			want: "claude-haiku-3",
		},
		{
			name: "falls back to sonnet",
			ids:  []string{"vendor-opus", "vendor-sonnet"},
			want: "vendor-sonnet",
		},
		{
			name: "falls back to first model",
			ids:  []string{"model-a", "model-b"},
			want: "model-a",
		},
		{
			name: "empty list",
			ids:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chooseProbeModel(tt.ids); got != tt.want {
				t.Fatalf("chooseProbeModel() = %q, want %q", got, tt.want)
			}
		})
	}
}
