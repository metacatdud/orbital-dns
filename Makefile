# Define WASM directories
WASM_DIR := web/wasm
WASM_OUT := orbital/web/orbital.wasm
WASM_EXEC := orbital/web/wasm_exec.js

# Version Vars
VERSION_TAG := $(shell git describe --tags --always)
VERSION_VERSION := $(shell git log --date=iso --pretty=format:"%cd" -1) $(VERSION_TAG)
VERSION_COMPILE := $(shell date +"%F %T %z") by $(shell go version)
VERSION_BRANCH  := $(shell git rev-parse --abbrev-ref HEAD)
VERSION_GIT_DIRTY := $(shell git diff --no-ext-diff 2>/dev/null | wc -l | awk '{print $1}')
VERSION_DEV_PATH:= $(shell pwd)

# Go Checkup
GOPATH ?= $(shell go env GOPATH)
GO111MODULE:=auto
export GO111MODULE
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
endif
PATH := ${GOPATH}/bin:$(PATH)
GCFLAGS=-gcflags="all=-trimpath=${GOPATH}"
LDFLAGS=-ldflags="-s -w -X 'main.Version=${VERSION_TAG}' -X 'main.Compile=${VERSION_COMPILE}' -X 'main.Branch=${VERSION_BRANCH}'"

GO = go

V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m➡\033[0m")

# Commands
.PHONY: all
all: | init deps

.PHONY: init
init: ; $(info $(M) Installing tools dependencies ...) @ ## Install tools dependencies
	@if ! pre-commit --version > /dev/null 2>&1; then \
		echo "pre-commit is not installed. Please install it using one of the following methods:"; \
		echo "- For Debian/Ubuntu-based systems: apt install pre-commit"; \
		echo "- For macOS (Homebrew): brew install pre-commit"; \
		echo "- For Python environments: pip install pre-commit"; \
		exit 1; \
	fi

	pre-commit install --install-hooks
	pre-commit install --hook-type commit-msg

	$Q $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$Q $(GO) install github.com/go-critic/go-critic/cmd/gocritic@latest
	$Q $(GO) install github.com/sqs/goreturns@latest

.PHONY: deps
deps: ; $(info $(M) Installing project dependencies ...) @ ## Install project dependencies
	$Q $(GO) mod tidy

.PHONY: test
test:  ; $(info $(M) Running unit tests ...)	@ ## Run unit tests
	$Q $(GO) test -v  -coverprofile coverage.out ./...

.PHONY: build
build: ; $(info $(M) Building executable...) @ ## Build program binary
	$Q echo "ver   : ${VERSION_TAG}"
	$Q echo "veriso: ${VERSION_VERSION}"
	$Q echo "vergo : ${VERSION_COMPILE}"
	$Q mkdir -p bin
	$Q $(GO) generate ./...
	$Q ret=0 && for d in $$($(GO) list -f '{{if (eq .Name "main")}}{{.ImportPath}}{{end}}' ./...); do \
		b=$$(basename $${d}) ; \
		$(GO) build ${LDFLAGS} ${GCFLAGS} -o bin/$${b} $$d || ret=$$? ; \
		echo "$(M) Build: bin/$${b}" ; \
		echo "$(M) Done!" ; \
	done ; exit $$ret

.PHONY: run
run: ; $(info $(M) Running dev build (on the fly) ...) @ ## Run intermediate builds
	$Q $(GO) run -race ./...

# WASM Compilation
.PHONY: wasm
wasm: ; $(info $(M) Compiling WASM dashboard ...) @
	GOOS=js GOARCH=wasm $(GO) build -o $(WASM_OUT) $(WASM_DIR)/main.go
	cp "$(shell go env GOROOT)/misc/wasm/wasm_exec.js" $(WASM_EXEC)

help:
	$Q echo "\Orbital makefile\n----------------"
	$Q grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
