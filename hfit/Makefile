# Hot Fixture Tool CLI - Build Configuration
# Copyright (c) 2025 Daniele Cruciani <daniele@smartango.com>
# 
# This file is part of the Hot Fixture Tool project.
# GitHub: https://github.com/danielecr/hot-fixture-tool
# 
# Licensed under the terms specified in the LICENSE file.

BINARY_NAME=hfit

.PHONY: build clean install

build:
	go build -o $(BINARY_NAME)

clean:
	go clean
	rm -f $(BINARY_NAME)

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

test:
	go test ./...