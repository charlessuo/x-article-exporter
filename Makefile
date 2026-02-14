TEST_URL := https://x.com/demo_author/article/1234567890123456789
PREVIEW_DIR := /tmp/x-article-preview

.PHONY: test test-article test-article-dark preview preview-dark fmt-docs

test:
	@go test -count=1 ./...

test-article:
	@go run main.go --output test-article $(TEST_URL)

test-article-dark:
	@go run main.go --dark --output test-article-dark $(TEST_URL)

preview: test-article
	@rm -f $(PREVIEW_DIR)/page-*.png
	@mkdir -p $(PREVIEW_DIR)
	@magick -density 150 test-article.pdf -quality 90 $(PREVIEW_DIR)/page-%d.png
	@echo "Preview: $(PREVIEW_DIR)/page-*.png ($$(ls $(PREVIEW_DIR)/page-*.png 2>/dev/null | wc -l | tr -d ' ') pages)"

preview-dark: test-article-dark
	@rm -f $(PREVIEW_DIR)/dark-page-*.png
	@mkdir -p $(PREVIEW_DIR)
	@magick -density 150 test-article-dark.pdf -quality 90 $(PREVIEW_DIR)/dark-page-%d.png
	@echo "Preview: $(PREVIEW_DIR)/dark-page-*.png ($$(ls $(PREVIEW_DIR)/dark-page-*.png 2>/dev/null | wc -l | tr -d ' ') pages)"

fmt-docs:
	@npx prettier --write "shaping/*.md"
