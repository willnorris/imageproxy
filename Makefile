NAME = pixie
COMMIT = $(shell git rev-parse --short HEAD 2> /dev/null || date '+%s')
VERSION = $(shell git describe 2> /dev/null || echo "0.0.0-$(COMMIT)")
BUILDTIME = $(shell date +%Y-%m-%dT%T%z)

DIST_OPTS = -a -tags netgo -installsuffix netgo
LD_OPTS = -ldflags="-X main.version=$(VERSION) -X main.buildtime=$(BUILDTIME) -w"

# BUILD_CMD = CGO_ENABLED=0 go build $(LD_OPTS)
BUILD_CMD = CGO_ENABLED=0 go build

SOURCE_FILES = ./cmd/$(NAME)/*.go

clean:
	rm -rf ./dist

dist:
	mkdir -p dist
	GOOS=linux $(BUILD_CMD) -o ./dist/$(NAME)-linux $(SOURCE_FILES)