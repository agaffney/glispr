# Determine root directory
ROOT_DIR=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

# Gather all .go files for use in dependencies below
GO_FILES=$(shell find $(ROOT_DIR) -name '*.go')

# Gather list of expected binaries
BINARIES=$(shell cd $(ROOT_DIR)/cmd && ls -1)

.PHONY: all clean

all: $(BINARIES)

clean:
	rm -f $(BINARIES)

$(BINARIES): $(GOFILES)
	go build -o $(ROOT_DIR)/$(@) $(ROOT_DIR)/cmd/$(@)

.PHONY: test

test:
	go test -v ./...
