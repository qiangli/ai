#!/bin/bash

function tidy() {
	go mod tidy
	go fmt ./...
	go vet ./...
}

function build_ai() {
	local os=$1
	local arch=$2

	local binary=ai-$os-$arch
	if [[ $os == "windows" ]]; then
		binary="${binary}.exe"
	fi

	CLI_FLAGS=""

	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o bin/$binary -ldflags="-w -extldflags '-static' $CLI_FLAGS" ./cmd/ai
}

##
tidy

for os in linux darwin windows; do
	for arch in amd64 arm64; do
		echo "Building for $os/$arch"
		build_ai $os $arch
	done
done