#!/bin/sh
go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.4
go mod download
