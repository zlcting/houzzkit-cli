#!/bin/bash
APP_NAME="houzzkit-cli"
OUTPUT_DIR="dist"

mkdir -p $OUTPUT_DIR

# Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $OUTPUT_DIR/$APP_NAME-linux-arm64 main.go
# macOS ARM64
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $OUTPUT_DIR/$APP_NAME-darwin-arm64 main.go
# macOS x86_64 (Intel)
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $OUTPUT_DIR/$APP_NAME-darwin-amd64 main.go
# Windows ARM64
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o $OUTPUT_DIR/$APP_NAME-windows-arm64.exe main.go

echo "编译完成！文件输出在 $OUTPUT_DIR 目录中。"