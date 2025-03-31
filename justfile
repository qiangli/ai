#!/usr/bin/env -S just --justfile

default:
  @just --list

generate:
    go generate ./...

build: generate
    ./build.sh

test:
    go test -short ./...

tidy:
    go mod tidy
    go fmt ./...
    go vet ./...

# Generate a git commit message
git-message:
    git diff origin/main | go run ./cmd/ai --quiet --format=text @git/long }

# git commit
git-commit: git-message
    git commit -m "$(pbpaste)"

# git commit --amend
git-amend: git-message
    git commit --amend -m "$(pbpaste)"

install: build test
    go install ./cmd/ai

# Update all dependencies
update:
    go get -u ./...
