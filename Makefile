# Copyright 2020 Changkun Ou. All rights reserved.
# Use of this source code is governed by a GNU GPL-3.0
# license that can be found in the LICENSE file.

BINARY = tli
TARGET = -o $(BINARY)
BUILD_FLAGS = $(TARGET) -mod=vendor

all:
	go build $(BUILD_FLAGS)
clean:
	rm tli
