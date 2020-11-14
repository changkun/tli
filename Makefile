# Copyright 2020 Changkun Ou. All rights reserved.
# Use of this source code is governed by a GNU GPL-3.0
# license that can be found in the LICENSE file.

VERSION = $(shell git describe --always --tags)
BUILDTIME = $(shell date +%FT%T%z)
GOPATH=$(shell go env GOPATH)
BINARY = tli
TARGET = -o $(BINARY)
BUILD_SETTINGS = -ldflags="-X main.Version=$(VERSION) -X main.BuildTime=$(BUILDTIME)"
BUILD_FLAGS = $(TARGET) $(BUILD_SETTINGS) -mod=vendor

all:
	go build $(BUILD_FLAGS)
clean:
	rm tli
