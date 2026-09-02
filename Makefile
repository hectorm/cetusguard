#!/usr/bin/make -f

SHELL := /bin/sh
.SHELLFLAGS := -euc

prefix ?= /usr/local
exec_prefix ?= $(prefix)
bindir ?= $(exec_prefix)/bin

GIT ?= git
GO ?= go
GO_TEST_ARGS ?=
GOFMT ?= gofmt
GOFMT_ARGS ?=
GOSEC ?= go run github.com/securego/gosec/v2/cmd/gosec@latest
GOSEC_ARGS ?=
MODERNIZE ?= go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest
MODERNIZE_ARGS ?=
STATICCHECK ?= go run honnef.co/go/tools/cmd/staticcheck@latest
STATICCHECK_ARGS ?=
GOVULNCHECK ?= go run golang.org/x/vuln/cmd/govulncheck@latest
GOVULNCHECK_ARGS ?=
INSTALL ?= install
INSTALL_PROGRAM = $(INSTALL)
INSTALL_DATA = $(INSTALL) -m 644

VERSION ?= unknown
ifeq ($(origin VERSION), file)
  GIT_TAG := $(shell '$(GIT)' tag -l --points-at HEAD 2>/dev/null | sed -n '1p')
  GIT_SHA := $(shell '$(GIT)' rev-parse HEAD 2>/dev/null | cut -c1-8)
  VERSION := $(or $(GIT_TAG),$(GIT_SHA),$(VERSION))
endif

GOOS := $(shell '$(GO)' env GOOS)
GOARCH := $(shell '$(GO)' env GOARCH)
GOVARIANT := $(GO386)$(GOAMD64)$(GOARM)$(GOMIPS)$(GOMIPS64)$(GOPPC64)

CC ?=
CGO_ENABLED ?= 0
CGO_CFLAGS ?=
CGO_LDFLAGS ?=

GOFLAGS ?=
override GOFLAGS += -trimpath

GO_LDFLAGS ?= -s -w
override GO_LDFLAGS += -X "$(shell '$(GO)' list -m)/internal/config.Version=$(VERSION)"

GO_TEST_LDFLAGS = $(GO_LDFLAGS)
ifneq ($(CGO_ENABLED),0)
  GO_TEST_LDFLAGS += -linkmode=external
endif

SRCS := $(shell '$(GIT)' ls-files '*.go' 2>/dev/null || find ./ -name '*.go' -not -path './dist/*')
EXEC := cetusguard-$(VERSION)-$(GOOS)-$(GOARCH)

ifneq ($(GOVARIANT),)
  EXEC := $(addsuffix -$(GOVARIANT), $(EXEC))
endif

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
	CGO_ENABLED='$(CGO_ENABLED)' CGO_CFLAGS='$(CGO_CFLAGS)' CGO_LDFLAGS='$(CGO_LDFLAGS)' CC='$(CC)' \
		'$(GO)' build $(GOFLAGS) -ldflags '$(GO_LDFLAGS)' -o '$@' ./cmd/cetusguard/

.PHONY: lint
lint: gofmt gosec modernize staticcheck

.PHONY: gofmt
gofmt:
	@test -z "$$($(GOFMT) -s -l $(GOFMT_ARGS) ./ | tee /dev/stderr)"

.PHONY: gosec
gosec:
	@$(GOSEC) -tests $(GOSEC_ARGS) ./...

.PHONY: modernize
modernize:
	@$(MODERNIZE) -test $(MODERNIZE_ARGS) ./...

.PHONY: staticcheck
staticcheck:
	@$(STATICCHECK) -tests -checks=all,-ST1000 $(STATICCHECK_ARGS) ./...

.PHONY: govulncheck
govulncheck:
	@$(GOVULNCHECK) -test $(GOVULNCHECK_ARGS) ./...

.PHONY: test check
test check:
	CGO_ENABLED='$(CGO_ENABLED)' CGO_CFLAGS='$(CGO_CFLAGS)' CGO_LDFLAGS='$(CGO_LDFLAGS)' CC='$(CC)' \
		'$(GO)' test $(GOFLAGS) -ldflags '$(GO_TEST_LDFLAGS)' -v -count=1 $(GO_TEST_ARGS) ./...

.PHONY: test-e2e
test-e2e:
	./e2e/run.sh

.PHONY: installdirs
installdirs:
	@mkdir -p '$(DESTDIR)$(bindir)'

.PHONY: install
install: ./dist/$(EXEC) installdirs
	$(INSTALL_PROGRAM) '$<' '$(DESTDIR)$(bindir)/cetusguard'
	@$(MAKE) --no-print-directory installcheck DESTDIR='$(DESTDIR)' bindir='$(bindir)'

.PHONY: install-strip
install-strip:
	@$(MAKE) INSTALL_PROGRAM='$(INSTALL_PROGRAM) -s' install

.PHONY: installcheck
installcheck:
	@test -x '$(DESTDIR)$(bindir)/cetusguard'

.PHONY: uninstall
uninstall:
	@rm -fv '$(DESTDIR)$(bindir)/cetusguard'

.PHONY: clean distclean
clean distclean:
	@rm -rfv './dist/'
