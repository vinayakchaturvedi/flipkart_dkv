
GOOS ?= darwin
GOARCH ?= amd64
CGO_ENABLED ?= 1
CGO_CFLAGS ?=
CGO_LDFLAGS ?=
BUILD_TAGS ?=
VERSION ?=
BIN_EXT ?=

GO := GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) CGO_CFLAGS=$(CGO_CFLAGS) CGO_LDFLAGS=$(CGO_LDFLAGS) GO111MODULE=on go

PACKAGES = $(shell $(GO) list ./... | grep -v '/vendor/')

PROTOBUFS = $(shell find . -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq | grep -v /vendor/)

TARGET_PACKAGES = $(shell find . -name 'main.go' -print0 | xargs -0 -n1 dirname | sort | uniq | grep -v /vendor/)

ifeq ($(VERSION),)
  VERSION = latest
endif
LDFLAGS = -ldflags "-X \"github.com/flipkart-incubator/dkv/version.Version=$(VERSION)\""

ifeq ($(GOOS),windows)
  BIN_EXT = .exe
endif

.DEFAULT_GOAL := build

.PHONY: protoc
protoc:
	@echo ">> generating proto code"
	@for proto_dir in $(PROTOBUFS); do echo $$proto_dir; protoc --proto_path=$$proto_dir --go_out=plugins=grpc:$$proto_dir $$proto_dir/*.proto || exit 1; done

.PHONY: format
format:
	@echo ">> formatting code"
	@$(GO) fmt $(PACKAGES)

.PHONY: test
test:
	@echo ">> testing all packages"
	@echo "   GOOS        = $(GOOS)"
	@echo "   GOARCH      = $(GOARCH)"
	@echo "   CGO_ENABLED = $(CGO_ENABLED)"
	@echo "   CGO_CFLAGS  = $(CGO_CFLAGS)"
	@echo "   CGO_LDFLAGS = $(CGO_LDFLAGS)"
	@echo "   BUILD_TAGS  = $(BUILD_TAGS)"
	@$(GO) test -v -tags="$(BUILD_TAGS)" $(PACKAGES)

.PHONY: build
build:
	@echo ">> building binaries"
	@echo "   GOOS        = $(GOOS)"
	@echo "   GOARCH      = $(GOARCH)"
	@echo "   CGO_ENABLED = $(CGO_ENABLED)"
	@echo "   CGO_CFLAGS  = $(CGO_CFLAGS)"
	@echo "   CGO_LDFLAGS = $(CGO_LDFLAGS)"
	@echo "   BUILD_TAGS  = $(BUILD_TAGS)"
	@echo "   VERSION     = $(VERSION)"
	@for target_pkg in $(TARGET_PACKAGES); do echo $$target_pkg; $(GO) build -tags="$(BUILD_TAGS)" $(LDFLAGS) -o ./bin/`basename $$target_pkg`$(BIN_EXT) $$target_pkg || exit 1; done

.PHONY: install
install:
	@echo ">> installing binaries"
	@echo "   GOOS        = $(GOOS)"
	@echo "   GOARCH      = $(GOARCH)"
	@echo "   CGO_ENABLED = $(CGO_ENABLED)"
	@echo "   CGO_CFLAGS  = $(CGO_CFLAGS)"
	@echo "   CGO_LDFLAGS = $(CGO_LDFLAGS)"
	@echo "   BUILD_TAGS  = $(BUILD_TAGS)"
	@echo "   VERSION     = $(VERSION)"
	@for target_pkg in $(TARGET_PACKAGES); do echo $$target_pkg; $(GO) install -tags="$(BUILD_TAGS)" $(LDFLAGS) $$target_pkg || exit 1; done

.PHONY: dist
dist:
	@echo ">> packaging binaries"
	@echo "   GOOS        = $(GOOS)"
	@echo "   GOARCH      = $(GOARCH)"
	@echo "   CGO_ENABLED = $(CGO_ENABLED)"
	@echo "   CGO_CFLAGS  = $(CGO_CFLAGS)"
	@echo "   CGO_LDFLAGS = $(CGO_LDFLAGS)"
	@echo "   BUILD_TAGS  = $(BUILD_TAGS)"
	@echo "   VERSION     = $(VERSION)"
	mkdir -p ./dist/$(GOOS)-$(GOARCH)/bin
	@for target_pkg in $(TARGET_PACKAGES); do echo $$target_pkg; $(GO) build -tags="$(BUILD_TAGS)" $(LDFLAGS) -o ./dist/$(GOOS)-$(GOARCH)/bin/`basename $$target_pkg`$(BIN_EXT) $$target_pkg || exit 1; done
	(cd ./dist/$(GOOS)-$(GOARCH); tar zcfv ../dkv-${VERSION}.$(GOOS)-$(GOARCH).tar.gz .)

.PHONY: git-tag
git-tag:
	@echo ">> tagging github"
	@echo "   VERSION = $(VERSION)"
ifeq ($(VERSION),$(filter $(VERSION),latest master ""))
	@echo "please specify VERSION"
else
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
endif

.PHONY: clean
clean:
	@echo ">> cleaning binaries"
	rm -rf ./bin
	rm -rf ./data
	rm -rf ./dist