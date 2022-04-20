#!/usr/bin/make -f

SHELL := /bin/sh
.SHELLFLAGS := -euc

DESTDIR ?=

prefix ?= /usr/local
exec_prefix ?= $(prefix)
bindir ?= $(exec_prefix)/bin

GIT := git
GO := go
GOFMT := gofmt
INSTALL := install

INSTALL_PROGRAM := $(INSTALL)
INSTALL_DATA := $(INSTALL) -m 644

GIT_TAG := $(shell '$(GIT)' tag -l --contains HEAD)
GIT_SHA := $(shell '$(GIT)' rev-parse HEAD)
VERSION := $(if $(GIT_TAG),$(GIT_TAG),$(GIT_SHA))

GOOS := $(shell '$(GO)' env GOOS)
GOARCH := $(shell '$(GO)' env GOARCH)
export CGO_ENABLED := 0

GOFLAGS := -trimpath
LDFLAGS := -s -w -X "main.version=$(VERSION)"

SRCS := $(shell '$(GIT)' ls-files '*.go' 2>/dev/null ||:)
EXEC := cetusguard-$(GOOS)-$(GOARCH)

ifeq ($(GOOS),windows)
	EXEC := $(addsuffix .exe, $(EXEC))
endif

.PHONY: all
all: build

.PHONY: build
build: ./dist/$(EXEC)

.PHONY: run
run: ./dist/$(EXEC)
	'$<'

./dist/$(EXEC): $(SRCS)
	@mkdir -p "$$(dirname '$@')"
	'$(GO)' build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o '$@' ./cmd/cetusguard/

.PHONY: lint
lint:
	@test -z "$$('$(GOFMT)' -s -l ./ | tee /dev/stderr)"

.PHONY: test
test:
	'$(GO)' test -v ./...

.PHONY: test-e2e
test-e2e:
	./e2e/run.sh

.PHONY: install
install:
	@mkdir -p '$(DESTDIR)$(bindir)'
	$(INSTALL_PROGRAM) './dist/$(EXEC)' '$(DESTDIR)$(bindir)/$(EXEC)'

PHONY: installcheck
installcheck:
	@test -x '$(DESTDIR)$(bindir)/$(EXEC)'

.PHONY: uninstall
uninstall:
	rm -fv '$(DESTDIR)$(bindir)/$(EXEC)'

.PHONY: clean
clean:
	rm -rfv './dist/'
