# workbuddy CLIProxyAPI plugin
#
# Build a shared library matching the current host:
#   make build
#
# Cross-ish local package (requires a C toolchain for GOOS/GOARCH):
#   make package VERSION=0.2.0 GOOS=linux GOARCH=amd64
#
# Plugin store asset name: workbuddy_<version>_<goos>_<goarch>.zip

PLUGIN_ID ?= workbuddy
VERSION   ?= 0.2.0
GOOS      ?= $(shell go env GOOS)
GOARCH    ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
  LIB_EXT := dll
else ifeq ($(GOOS),darwin)
  LIB_EXT := dylib
else
  LIB_EXT := so
endif

LIB_NAME     := $(PLUGIN_ID).$(LIB_EXT)
ARCHIVE_NAME := $(PLUGIN_ID)_$(VERSION)_$(GOOS)_$(GOARCH).zip
DIST_DIR     := dist
LDFLAGS      := -s -w -X main.pluginVersion=$(VERSION)

.PHONY: build package clean vet test list

list:
	@echo "PLUGIN_ID=$(PLUGIN_ID)"
	@echo "VERSION=$(VERSION)"
	@echo "GOOS=$(GOOS) GOARCH=$(GOARCH)"
	@echo "LIB_NAME=$(LIB_NAME)"
	@echo "ARCHIVE_NAME=$(ARCHIVE_NAME)"

build:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) \
	  go build -trimpath -buildmode=c-shared \
	    -ldflags "$(LDFLAGS)" \
	    -o $(DIST_DIR)/$(LIB_NAME) .
	@rm -f $(DIST_DIR)/$(PLUGIN_ID).h $(DIST_DIR)/*.h

package: build
	go run ./.github/scripts/package-release.go \
	  -library $(DIST_DIR)/$(LIB_NAME) \
	  -archive $(ARCHIVE_NAME) \
	  -checksum $(ARCHIVE_NAME).sha256
	@echo "packed $(ARCHIVE_NAME)"

vet:
	go vet ./...

# No unit tests yet; keep the target for CI symmetry with other CPA plugins.
test:
	@echo "no tests"

clean:
	rm -rf $(DIST_DIR) $(PLUGIN_ID)_*.zip $(PLUGIN_ID)_*.zip.sha256 checksums.txt *.so *.dylib *.dll *.h
