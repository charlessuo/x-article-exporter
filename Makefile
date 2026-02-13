.PHONY: test-article fmt-docs

test-article:
	@go run main.go https://x.com/demo_author/article/1234567890123456789

fmt-docs:
	@npx prettier --write "shaping/*.md"
