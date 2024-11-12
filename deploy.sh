#!/bin/sh

export GOOS="linux"
export GOARCH="arm64"

go build -o arm_hello . && \
scp arm_hello nanopi:
