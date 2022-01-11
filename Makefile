Dirs=$(shell ls)
COMMIT_ID ?= $(shell git rev-parse --short HEAD || echo "0.0.0")

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifneq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

default:  build


GORELEASER_BIN = $(shell pwd)/bin/goreleaser
install-goreleaser: ## check license if not exist install go-lint tools
	$(call go-get-tool,$(GORELEASER_BIN),github.com/goreleaser/goreleaser@latest)


build: SHELL:=/bin/bash
build: install-goreleaser clean ## build binaries by default
	@echo "build endpoints-balance bin"
	$(GORELEASER_BIN) build --snapshot --rm-dist  --timeout=1h

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

clean: ## clean
	rm -rf dist

ADDLICENSE_BIN = $(shell pwd)/bin/addlicense
install-addlicense: ## check license if not exist install go-lint tools
	$(call go-get-tool,$(ADDLICENSE_BIN),github.com/google/addlicense@latest)

filelicense:
filelicense: install-addlicense
	for file in ${Dirs} ; do \
		if [[  $$file != '_output' && $$file != 'docs' && $$file != 'vendor' && $$file != 'logger' && $$file != 'applications' ]]; then \
			$(ADDLICENSE_BIN)  -y $(shell date +"%Y") -c "cuisongliu@qq.com." -f hack/LICENSE ./$$file ; \
		fi \
    done
