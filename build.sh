#!/bin/bash
set -xue

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

	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o "bin/$binary" -ldflags="-w -extldflags '-static' $CLI_FLAGS" ./cmd

	ln -sf "$binary" "bin/ai"
}

function build_all() {
	# local os_list=("linux" "darwin" "windows")
	local os_list=("linux" "darwin")
	local arch_list=("arm64", "amd64")

	for os in "${os_list[@]}"; do
		for arch in "${arch_list[@]}"; do
			echo "Building for $os/$arch"
			build_ai "$os" "$arch"
			echo "Build completed for $os/$arch"
		done
	done
}

function build() {
	local os
	local arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	if [[ "$arch" == "x86_64" ]]; then                                                            
      arch="amd64"                                                                                
    fi                                                                                            

	build_ai "${os}" "${arch}"
	echo "Build completed for ${os} ${arch}"
}

##
tidy
#
opt=${1:""}
if [[ $opt == "all" ]]; then
	build_all
else
	build
fi

