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
	@build.sh

test:  ## Test
	@go test -short ./...

commit-message: ## Generate commit message and copy the message to clipboard
	@git diff origin main|go run ./cmd/ai --dry-run=false @ask write commit message for git =+

install: build ## Install
	@go install ./cmd/ai

.PHONY: build test install

###
