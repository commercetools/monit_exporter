# ###########
# Global Vars

export PATH := ./bin:$(PATH)

APP_NAME ?= monit-exporter
VERSION := $(shell cat ./VERSION)
ENV := production
BIN_NAME := ./bin/$(APP_NAME)

CPWD := $(PWD)

TMP_DIRS := ./bin
TMP_DIRS += ./dist

GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_DESCRIBE := $(shell git describe --tags --always)

GOOS := linux
GOARCH := amd64

LDFLAGS :=
LDFLAGS += -X main.VersionCommit=$(GIT_COMMIT)
LDFLAGS += -X main.VersionTag=$(GIT_DESCRIBE)
LDFLAGS += -X main.VersionFull=$(VERSION)
LDFLAGS += -X main.VersionEnv=$(ENV)

RELEASE_VERSION ?= latest

# ##################
# Makefile functions

define deps_tag
	@if [[ "$(message)"x == "x" ]]; then \
		echo -e "\n Error: the commit message was not provided."; \
		$(call show_usage) \
		exit 1; \
	fi
endef

# Build a beta version
.PHONY: build
build:
	@test -d ./bin || mkdir ./bin
	GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build \
		-ldflags "$(LDFLAGS)" \
		$(BUILD_TAGS) \
		-o $(BIN_NAME) && \
		strip $(BIN_NAME) && \
		sha256sum $(BIN_NAME) |awk '{print$1}' > $(BIN_NAME).sha256

.PHONY: run
run:
	$(BIN_NAME)

.PHONY: version
version: build
	$(BIN_NAME) -v

.PHONY: clean
clean:
	@rm -f bin/$(BIN_NAME)*

# ###############
# GHR
# GitHub releaser
# https://github.com/tcnksm/ghr
deps-install-ghr:
	go get -u github.com/tcnksm/ghr

# #######
# Release
tag:
	$(call deps_tag,$@)
	git tag -a $(shell cat VERSION) -m "$(message)"
	git push origin $(shell cat VERSION)

# Release builder
release: build
	ghr $(RELEASE_VERSION) bin/

release-master: build
	ghr --recreate $(RELEASE_VERSION) bin/

# #########
# Tshooting

# Analyze bin sizes
check-bin:
	$(shell eval `go build -work -a 2>&1` && find $$WORK -type f -name "*.a" |xargs -I{} du -hxs "{}" | sort -rh | sed -e s:${WORK}/::g )

# Check static libs on the binary
check-size:
	go tool nm -sort size -size monit_exporter

packer-bin:
	upx --brute monit_exporter
