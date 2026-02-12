package extract

import "testing"

func TestExtractArticleID(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{
			name: "valid x.com URL",
			url:  "https://x.com/i/article/1234567890",
			want: "1234567890",
		},
		{
			name: "valid twitter.com URL",
			url:  "https://twitter.com/i/article/9876543210",
			want: "9876543210",
		},
		{
			name: "valid URL with trailing slash",
			url:  "https://x.com/i/article/1234567890/",
			want: "1234567890",
		},
		{
			name: "valid URL with query params",
			url:  "https://x.com/i/article/1234567890?ref=home",
			want: "1234567890",
		},
		{
			name: "valid URL with username",
			url:  "https://x.com/demo_author/article/1234567890123456789",
			want: "1234567890123456789",
		},
		{
			name: "valid twitter.com URL with username",
			url:  "https://twitter.com/someuser/article/9876543210",
			want: "9876543210",
		},
		{
			name:    "wrong host",
			url:     "https://example.com/i/article/123",
			wantErr: true,
		},
		{
			name:    "tweet URL instead of article",
			url:     "https://x.com/user/status/123456",
			wantErr: true,
		},
		{
			name:    "missing article ID",
			url:     "https://x.com/i/article/",
			wantErr: true,
		},
		{
			name:    "non-numeric article ID",
			url:     "https://x.com/i/article/abc",
			wantErr: true,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "just a path",
			url:     "/i/article/123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractArticleID(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExtractArticleID(%q) = %q, want error", tt.url, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractArticleID(%q) error = %v", tt.url, err)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractArticleID(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
