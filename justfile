#!/usr/bin/env -S just --justfile

default:
  @just --list

build:
    ./build.sh

build-all: tidy
    ./build.sh all

test:
    go test -short ./...

# # Start hub services with 'ask' agent in debug mode (verbose)
# hub flag_args='':
#     ai /hub start --address ":58080" --pg-address ":25432" --mysql-address ":3306" --redis-address ":6379" --llm-proxy-address ":8000" {{flag_args}}

tidy:
    go mod tidy
    go fmt ./...
    go vet ./...

# Generate a git commit message
git-message:
    git diff origin/main | go run ./cmd --quiet --format=text @git/long }

# git commit
git-commit: git-message
    git commit -m "$(pbpaste)"

# git commit --amend
git-amend: git-message
    git commit --amend -m "$(pbpaste)"

install: build test
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$(go env GOPATH)/bin/ai" -ldflags="-w -extldflags '-static' ${CLI_FLAGS:-}" ./cmd

# Update all dependencies
update:
    go get -u ./...

clean-cache:
    go clean -modcache
