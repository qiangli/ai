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

git-message:
    git diff origin/main | go run ./cmd/ai --dry-run=false @git/conventional =+

git-commit: git-message
    git commit -m "$(pbpaste)"

git-amend: git-message
    git commit --amend -m "$(pbpaste)"

install: build test
    go install ./cmd/ai