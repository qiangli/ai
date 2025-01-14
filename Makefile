#
# make ai
#

###
.DEFAULT_GOAL := help

.PHONY: help
help: Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

##
build: ## Build
	@./build.sh

test:  ## Test
	@go test -short ./...

tidy: ## Tidy
	@go mod tidy && go fmt ./...

git-message: ## Generate commit message and copy the message to clipboard
	@git diff origin/main|go run ./cmd/ai --dry-run=false @git =+

git-commit: git-message ## Generate and commit with the message
	@git commit -m "$$(pbpaste)"

install: build test ## Install
	@go install ./cmd/ai

.PHONY: build test install commit-message

###
