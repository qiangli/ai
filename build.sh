#!/bin/bash

go mod tidy
go fmt ./...
go vet ./...

go build -o bin/ai ./cmd/ai

