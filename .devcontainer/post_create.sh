#!/bin/sh
go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
go install github.com/go-task/task/v3/cmd/task@v3.50.0
go mod download
npm install --prefix web/app
