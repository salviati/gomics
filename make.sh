#!/bin/sh
go build -ldflags "-s -w -X main.buildDate=$(date +%Y%m%d) -X main.gitVersion=$(git rev-parse HEAD)" -o gomics .
#upx --best gomics
