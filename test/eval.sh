#!/usr/bin/env ai /sh:bash --format raw --base ./test/data/ --script

### Build

# bash
#/usr/bin/env ai /sh:bash --script
# time /bin/bash ./build.sh
# 

### Test

# # bash
# #/usr/bin/env ai /sh:bash --script
# set -xe
# go test -short ./...
# echo "EXIT STATUS: $?"

# # 

### All

# Buid, install, unit tests, and bash integration tests

# ---
# dependencies:
#   - tidy
#   - build
#   - test
#   - install
# ---

# # bash
# #/usr/bin/env ai /sh:bash --script
# set -xe
# time ./test/all.sh
# echo "EXIT STATUS: $?"
# # 

### Tidy

# # bash
# #/usr/bin/env ai /sh:bash --script
# ##
# set -xe
# go mod tidy
# go fmt ./...
# go vet ./...
# echo "EXIT STATUS: $?"

# # 

### Install

# # # bash
# # #/usr/bin/env ai /sh:bash --script
# set -xe
# time CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "$(go env GOPATH)/bin/ai" -ldflags="-w -extldflags '-static' ${CLI_FLAGS:-}" ./cmd
# echo "EXIT STATUS: $?"
# # # 

### Update

# Update all dependencies

# # bash
# #/usr/bin/env ai /sh:bash --script
# go get -u ./...
# echo "success"
# # 

### Clean Cache

# # bash
# #/usr/bin/env ai /sh:bash --script
# go clean -modcache
# # 


###