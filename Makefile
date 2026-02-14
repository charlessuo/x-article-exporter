TEST_URL := https://x.com/demo_author/article/1234567890123456789
PREVIEW_DIR := /tmp/x-article-preview

.PHONY: test test-full test-article preview fmt-docs

test:
	@go test -short ./...

test-full:
	@go test -count=1 ./...

test-article:
	@go run main.go --output test-article $(TEST_URL)

preview: test-article
	@rm -f $(PREVIEW_DIR)/page-*.png
	@mkdir -p $(PREVIEW_DIR)
	@magick -density 150 test-article.pdf -quality 90 $(PREVIEW_DIR)/page-%d.png
	@echo "Preview: $(PREVIEW_DIR)/page-*.png ($$(ls $(PREVIEW_DIR)/page-*.png 2>/dev/null | wc -l | tr -d ' ') pages)"

fmt-docs:
	@npx prettier --write "shaping/*.md"
