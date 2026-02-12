package extract

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractArticleID parses an X article URL and returns the snowflake article ID.
// Accepts URLs like:
//   - https://x.com/i/article/1234567890
//   - https://x.com/{username}/article/1234567890
//   - https://twitter.com/{username}/article/1234567890
func ExtractArticleID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(u.Hostname())
	if host != "x.com" && host != "twitter.com" {
		return "", fmt.Errorf("invalid article URL: host must be x.com or twitter.com, got %q", host)
	}

	// Expected path: /{username_or_i}/article/{id}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 3 || parts[1] != "article" {
		return "", fmt.Errorf("invalid article URL: expected path /{user}/article/{id}, got %q", u.Path)
	}

	articleID := parts[2]
	if articleID == "" {
		return "", fmt.Errorf("invalid article URL: empty article ID")
	}

	// Snowflake IDs are numeric
	for _, c := range articleID {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("invalid article URL: article ID must be numeric, got %q", articleID)
		}
	}

	return articleID, nil
}
