PROJECT_NAME=fabric8-common
PACKAGE_NAME := github.com/fabric8-services/$(PROJECT_NAME)
CUR_DIR=$(shell pwd)
TMP_PATH=$(CUR_DIR)/tmp
INSTALL_PREFIX=$(CUR_DIR)/bin
VENDOR_DIR=vendor
SOURCE_DIR ?= .
SOURCES := $(shell find $(SOURCE_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)

# declares variable that are OS-sensitive
ifeq ($(OS),Windows_NT)
include ./.make/Makefile.win
else
include ./.make/Makefile.lnx
endif
include ./.make/test.mk

GIT_BIN := $(shell command -v $(GIT_BIN_NAME) 2> /dev/null)
DEP_BIN_DIR := $(TMP_PATH)/bin
# DEP_BIN := $(DEP_BIN_DIR)/$(DEP_BIN_NAME)
DEP_BIN := $(shell command -v $(DEP_BIN_NAME) 2> /dev/null)
DEP_VERSION=v0.4.1
GO_BIN := $(shell command -v $(GO_BIN_NAME) 2> /dev/null)

DOCKER_BIN := $(shell command -v $(DOCKER_BIN_NAME) 2> /dev/null)
ifneq ($(OS),Windows_NT)
ifdef DOCKER_BIN
include ./.make/docker.mk
endif
endif


# This is a fix for a non-existing user in passwd file when running in a docker
# container and trying to clone repos of dependencies
GIT_COMMITTER_NAME ?= "user"
GIT_COMMITTER_EMAIL ?= "user@example.com"
export GIT_COMMITTER_NAME
export GIT_COMMITTER_EMAIL

COMMIT=$(shell git rev-parse HEAD)
GITUNTRACKEDCHANGES := $(shell git status --porcelain --untracked-files=no)
ifneq ($(GITUNTRACKEDCHANGES),)
COMMIT := $(COMMIT)-dirty
endif
BUILD_TIME=`date -u '+%Y-%m-%dT%H:%M:%SZ'`

.DEFAULT_GOAL := all

# If nothing was specified, run all targets as if in a fresh clone
.PHONY: all
## Default target - fetch dependencies and build.
all: prebuild-check deps generate build

.PHONY: format-go-code
## Formats any go file that differs from gofmt's style
format-go-code: prebuild-check
	@gofmt -s -l -w ${GOFORMAT_FILES}
	
.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$1, 0, index($$1, ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
			printf "%-35s -> %s\n", helpCommand, helpMessage; \
			lastLine = "" \
		} \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)

GOFORMAT_FILES := $(shell find  . -name '*.go' | grep -vEf .gofmt_exclude)

.PHONY: check-go-format
## Exists with an error if there are files whose formatting differs from gofmt's
check-go-format: prebuild-check
	@gofmt -s -l ${GOFORMAT_FILES} 2>&1 \
		| tee /tmp/gofmt-errors \
		| read \
	&& echo "ERROR: These files differ from gofmt's style (run 'make format-go-code' to fix this):" \
	&& cat /tmp/gofmt-errors \
	&& exit 1 \
	|| true

.PHONY: deps
## Download build dependencies.
deps: $(DEP_BIN) $(VENDOR_DIR)

# install dep in a the tmp/bin dir of the repo
# $(DEP_BIN):
# 	@echo "Installing 'dep' $(DEP_VERSION) at '$(DEP_BIN_DIR)'..."
# 	mkdir -p $(DEP_BIN_DIR)
# ifeq ($(UNAME_S),Darwin)
# 	@curl -L -s https://github.com/golang/dep/releases/download/$(DEP_VERSION)/dep-darwin-amd64 -o $(DEP_BIN) 
# 	@cd $(DEP_BIN_DIR) && \
# 	echo "c0d875504ddc69533200b61e72064e00fb37717a3aa36722634789a778f00e59  dep" > dep-darwin-amd64.sha256 && \
# 	shasum -a 256 --check dep-darwin-amd64.sha256
# else
# 	@curl -L -s https://github.com/golang/dep/releases/download/$(DEP_VERSION)/dep-linux-amd64 -o $(DEP_BIN)
# 	@cd $(DEP_BIN_DIR) && \
# 	echo "31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315  dep" > dep-linux-amd64.sha256 && \
# 	sha256sum -c dep-linux-amd64.sha256
# endif
# 	@chmod +x $(DEP_BIN)

$(VENDOR_DIR): Gopkg.toml Gopkg.lock
	@echo "checking dependencies..."
	@$(DEP_BIN) ensure -v 

.PHONY: analyze-go-code
## Run a complete static code analysis using the following tools: golint, gocyclo and go-vet.
analyze-go-code: golint gocyclo govet

# Build go tool to analysis the code
$(GOLINT_BIN):
	cd $(VENDOR_DIR)/github.com/golang/lint/golint && go build -v

## Run gocyclo analysis over the code.
golint: $(GOLINT_BIN)
	$(info >>--- RESULTS: GOLINT CODE ANALYSIS ---<<)
	@$(foreach d,$(GOANALYSIS_DIRS),$(GOLINT_BIN) $d 2>&1 | grep -vEf .golint_exclude || true;)

$(GOCYCLO_BIN):
	cd $(VENDOR_DIR)/github.com/fzipp/gocyclo && go build -v

## Run gocyclo analysis over the code.
gocyclo: $(GOCYCLO_BIN)
	$(info >>--- RESULTS: GOCYCLO CODE ANALYSIS ---<<)
	@$(foreach d,$(GOANALYSIS_DIRS),$(GOCYCLO_BIN) -over 10 $d | grep -vEf .golint_exclude || true;)

## Run go vet analysis over the code.
govet:
	$(info >>--- RESULTS: GO VET CODE ANALYSIS ---<<)
	@$(foreach d,$(GOANALYSIS_DIRS),go tool vet --all $d/*.go 2>&1;)

.PHONY: build
## build all packages
build: deps generate
	@echo "building all packages..."
	go build ./...

.PHONY: generate
generate: generate-mocks migration/sqlbindata_test.go

.PHONY: import
## import a pkg or a file from another repository, along with the commit history
# credit: http://www.pixelite.co.nz/article/extracting-file-folder-from-git-repository-with-full-git-history/
import: import-multiple-commits build

.PHONY: import-multiple-commits
import-multiple-commits:
# export the commits (as patches) from the source repo
	@echo "exporting content and commit history of pkg or file '$(pkg)' from '$(project)'..." 
	@cd $(GOPATH)/src/github.com/fabric8-services/$(project) 1>/dev/null && \
	git log --pretty=email --patch-with-stat --reverse --full-index --binary -- $(pkg) > /tmp/migrate.patch
# replace imports of root pkg and subpkgs
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 
# import the commits into the target repo
	@echo "importing pkg or file '$(pkg)' with commit history into `pwd`"
	@git am /tmp/migrate.patch 

.PHONY: import-commit
## import a pkg or a file from another repository, along with the commit history
import-commit: import-single-commit build

.PHONY: import-single-commit
import-single-commit:
# export the commits (as patches) from the source repo
	@echo "exporting content and log of '$(hash)' from '$(project)'..." 
	@cd $(GOPATH)/src/github.com/fabric8-services/$(project) 1>/dev/null && \
	git show --pretty=email --patch-with-stat --reverse --full-index --binary $(hash) > /tmp/migrate.patch
# replace imports of root pkg and subpkgs
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 
# import the commits into the target repo
	@echo "importing content and log of '$(hash)' with commit history into `pwd`"
	@git am /tmp/migrate.patch 

.PHONY: import-file-in-commit
## import a pkg or a file from another repository, along with the commit history
import-file-in-commit: import-single-commit-file build

.PHONY: import-single-commit-file
import-single-commit-file:
# export the commits (as patches) from the source repo
	@echo "exporting content and log of '$(file)' in '$(hash)' from '$(project)'..." 
	@cd $(GOPATH)/src/github.com/fabric8-services/$(project) 1>/dev/null && \
	git show --pretty=email --patch-with-stat --reverse --full-index --binary $(hash)~1..$(hash) -- $(file) > /tmp/migrate.patch
# replace imports of root pkg and subpkgs
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 
# import the commits into the target repo
	@echo "importing content and log of '$(file)' in '$(hash)' with commit history into `pwd`"
	@git am /tmp/migrate.patch 


$(INSTALL_PREFIX):
# Build artifacts dir
	mkdir -p $(INSTALL_PREFIX)

$(TMP_PATH):
	mkdir -p $(TMP_PATH)

.PHONY: show-info
show-info:
	$(call log-info,"$(shell go version)")
	$(call log-info,"$(shell go env)")

.PHONY: prebuild-check
prebuild-check: $(TMP_PATH) $(INSTALL_PREFIX) show-info
# Check that all tools where found
ifndef GIT_BIN
	$(error The "$(GIT_BIN_NAME)" executable could not be found in your PATH)
endif
ifndef DEP_BIN
	$(error The "$(DEP_BIN_NAME)" executable could not be found in your PATH)
endif

migration/sqlbindata_test.go: $(GO_BINDATA_BIN) $(wildcard migration/sql-test-files/*.sql)
	$(GO_BINDATA_BIN) \
		-o migration/sqlbindata_test.go \
		-pkg migration_test \
		-prefix migration/sql-test-files \
		-nocompress \
		migration/sql-test-files

$(GO_BINDATA_BIN): $(VENDOR_DIR)
	cd $(VENDOR_DIR)/github.com/jteeuwen/go-bindata/go-bindata && go build -v

# For the global "clean" target all targets in this variable will be executed
CLEAN_TARGETS =

CLEAN_TARGETS += clean-artifacts
.PHONY: clean-artifacts
## Removes the ./bin directory.
clean-artifacts:
	-rm -rf $(INSTALL_PREFIX)

CLEAN_TARGETS += clean-object-files
.PHONY: clean-object-files
## Runs go clean to remove any executables or other object files.
clean-object-files:
	go clean ./...

CLEAN_TARGETS += clean-generated
.PHONY: clean-generated
## Removes all generated code.
clean-generated:
	-rm -f ./migration/sqlbindata_test.go
	-rm -f ./test/generated/token/manager_configuration.go

CLEAN_TARGETS += clean-vendor
.PHONY: clean-vendor
## Removes the ./vendor directory.
clean-vendor:
	-rm -rf $(VENDOR_DIR)

# Keep this "clean" target here at the bottom
.PHONY: clean
## Runs all clean-* targets.
clean: $(CLEAN_TARGETS)
