#
# make ai
#

###
.DEFAULT_GOAL := help

.PHONY: help
help: Makefile
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

##
generate:
	@go generate ./...

build: generate ## Build
	@./build.sh

test:  ## Test
	@go test -short ./...

tidy: ## Tidy
	@go mod tidy && go fmt ./... && go vet ./...

git-message: ## Generate commit message and copy the message to clipboard
	@git diff origin/main|go run ./cmd/ai --dry-run=false @git/conventional =+

git-commit: git-message ## Generate and commit with the message
	@git commit -m "$$(pbpaste)"

git-amend: git-message ## Generate and amend with the message
	@git commit --amend -m "$$(pbpaste)"

install: build test ## Install
	@go install ./cmd/ai

.PHONY: build test install commit-message

###
