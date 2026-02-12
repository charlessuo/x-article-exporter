.PHONY: test-article
test-article:
	@go run main.go -auth-token "${AUTH_TOKEN}" -ct0 "${CT0}" https://x.com/demo_author/article/1234567890123456789
