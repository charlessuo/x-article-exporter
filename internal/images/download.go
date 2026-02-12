package images

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/annismckenzie/x-article-exporter/internal/model"
)

// DownloadImages fetches all images referenced in the article's entity map
// and the cover image, replacing URLs with base64 data URLs in-place.
func DownloadImages(ctx context.Context, article *model.Article) error {
	for key, entity := range article.EntityMap {
		if entity.Type != "IMAGE" && entity.Type != "MEDIA" {
			continue
		}
		src, ok := entity.Data["src"].(string)
		if !ok || src == "" {
			continue
		}
		dataURL, err := fetchAsDataURL(ctx, src)
		if err != nil {
			return fmt.Errorf("downloading image %s: %w", src, err)
		}
		entity.Data["src"] = dataURL
		article.EntityMap[key] = entity
	}

	if article.CoverImageURL != "" {
		dataURL, err := fetchAsDataURL(ctx, article.CoverImageURL)
		if err != nil {
			return fmt.Errorf("downloading cover image: %w", err)
		}
		article.CoverImageURL = dataURL
	}

	return nil
}

func fetchAsDataURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		mime = "image/jpeg"
	}
	if idx := strings.Index(mime, ";"); idx != -1 {
		mime = strings.TrimSpace(mime[:idx])
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mime, encoded), nil
}
