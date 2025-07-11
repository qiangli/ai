#!/usr/bin/env -S just --justfile

default:
  @just --list

generate:
    go generate ./...

build:
    ./build.sh

build-all: tidy generate
    ./build.sh all

test:
    go test -short ./...

# Start hub services with 'ask' agent in debug mode (verbose)
hub flag_args='':
    ai --agent ask --verbose --hub --hub-address ":58080" --hub-pg-address ":25432" --hub-mysql-address ":3306" --hub-redis-address ":6379" {{flag_args}}

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
    go build -o "$(go env GOPATH)/bin/ai" ./cmd

# Update all dependencies
update:
    go get -u ./...
