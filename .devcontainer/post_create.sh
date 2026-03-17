#!/bin/sh
go install github.com/air-verse/air@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go mod download
