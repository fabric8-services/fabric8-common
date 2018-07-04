CUR_DIR=$(shell pwd)
TMP_PATH=$(CUR_DIR)/tmp
INSTALL_PREFIX=$(CUR_DIR)/bin
VENDOR_DIR=vendor
SOURCE_DIR ?= .
SOURCES := $(shell find $(SOURCE_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)
DESIGN_DIR=design
DESIGNS := $(shell find $(SOURCE_DIR)/$(DESIGN_DIR) -path $(SOURCE_DIR)/vendor -prune -o -name '*.go' -print)

# declares variable that are OS-sensitive
SELF_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
ifeq ($(OS),Windows_NT)
include $(SELF_DIR)Makefile.win
else
include $(SELF_DIR)Makefile.lnx
endif

GIT_BIN := $(shell command -v $(GIT_BIN_NAME) 2> /dev/null)
DEP_BIN := $(shell command -v $(DEP_BIN_NAME) 2> /dev/null)
GO_BIN := $(shell command -v $(GO_BIN_NAME) 2> /dev/null)

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

.DEFAULT_GOAL := help

.PHONY: help
# Based on https://gist.github.com/rcmachado/af3db315e31383502660
## display this help text.
help:/
	$(info Available targets)
	$(info -----------------)
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		helpCommand = substr($$(pkg), 0, index($$(pkg), ":")-1); \
		if (helpMessage) { \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub(/##/, "\n                                     ", helpMessage); \
		} else { \
			helpMessage = "(No documentation)"; \
		} \
		printf "%-35s -> %s\n", helpCommand, helpMessage; \
		lastLine = "" \
	} \
	{ hasComment = match(lastLine, /^## (.*)/); \
          if(hasComment) { \
            lastLine=lastLine$$0; \
	  } \
          else { \
	    lastLine = $$0 \
          } \
        }' $(MAKEFILE_LIST)


.PHONY: deps
## fetch vendor dependencies
deps:
	@echo "fetching dependencies..."
	dep ensure -v

.PHONY: build
## build all packages
build: deps
	@echo "building all packages..."
	go build ./...

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
	
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
# replace imports of root pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
# rename imports of sub pkg
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
	
	@echo "converting goimports from \"github.com/fabric8-services/$(project)\" to \"github.com/fabric8-services/fabric8-common\"..."
# replace imports of root pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\"/\"github.com\/fabric8-services\/fabric8-common\"/g") 
	@eval sed -i -e $(SED_REGEX) /tmp/migrate.patch
# rename imports of sub pkg
	@$(eval SED_REGEX:="s/\"github.com\/fabric8-services\/$(project)\/\([a-zA-Z0-9/]*\)\"/\"github.com\/fabric8-services\/fabric8-common\/\1\"/g")
	@sed -i -e $(SED_REGEX) /tmp/migrate.patch 

# import the commits into the target repo
	@echo "importing commit '$(hash)' with commit history into `pwd`"
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


#-------------------------------------------------------------------------------
# Normal test targets
#
# These test targets are the ones that will be invoked from the outside. If
# they are called and the artifacts already exist, then the artifacts will
# first be cleaned and recreated. This ensures that the tests are always
# executed.
#-------------------------------------------------------------------------------

# By default reduce the amount of log output from tests
F8_LOG_LEVEL ?= error

# Output directory for coverage information
COV_DIR = $(TMP_PATH)/coverage

# Files that combine package coverages for unit- and integration-tests separately
COV_PATH_UNIT = $(TMP_PATH)/coverage.unit.mode-$(COVERAGE_MODE)
COV_PATH_INTEGRATION = $(TMP_PATH)/coverage.integration.mode-$(COVERAGE_MODE)

# File that stores overall coverge for all packages and unit- integration- and remote-tests
COV_PATH_OVERALL = $(TMP_PATH)/coverage.mode-$(COVERAGE_MODE)

# This pattern excludes some folders from the coverage calculation (see grep -v)
ALL_PKGS_EXCLUDE_PATTERN = "vendor\|account\/tenant\|app\'\|tool\/cli\|design\|client\|test"

# This pattern excludes some folders from the go code analysis
GOANALYSIS_PKGS_EXCLUDE_PATTERN="vendor|account/tenant|app|client|tool/cli"
GOANALYSIS_DIRS=$(shell go list -f {{.Dir}} ./... | grep -v -E $(GOANALYSIS_PKGS_EXCLUDE_PATTERN))


.PHONY: test-all
## Runs test-unit and test-integration targets.
test-all: prebuild-check test-unit test-integration test-remote

.PHONY: test-unit
## Runs the unit tests and produces coverage files for each package.
test-unit: prebuild-check clean-coverage-unit $(COV_PATH_UNIT)

.PHONY: test-unit-no-coverage
## Runs the unit tests and WITHOUT producing coverage files for each package.
test-unit-no-coverage: prebuild-check $(SOURCES)
	$(call log-info,"Running test: $@")
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	F8_DEVELOPER_MODE_ENABLED=1 F8_RESOURCE_UNIT_TEST=1 F8_LOG_LEVEL=$(F8_LOG_LEVEL) go test $(GO_TEST_VERBOSITY_FLAG) $(TEST_PACKAGES)

.PHONY: test-unit-no-coverage-junit
test-unit-no-coverage-junit: prebuild-check ${GO_JUNIT_BIN} ${TMP_PATH}
	bash -c "set -o pipefail; make test-unit-no-coverage 2>&1 | tee >(${GO_JUNIT_BIN} > ${TMP_PATH}/junit.xml)"

.PHONY: test-integration
## Runs the integration tests and produces coverage files for each package.
## Make sure you ran "make integration-test-env-prepare" before you run this target.
test-integration: prebuild-check clean-coverage-integration migrate-database $(COV_PATH_INTEGRATION)

.PHONY: test-integration-no-coverage
## Runs the integration tests WITHOUT producing coverage files for each package.
## Make sure you ran "make integration-test-env-prepare" before you run this target.
test-integration-no-coverage: prebuild-check migrate-database $(SOURCES)
	$(call log-info,"Running test: $@")
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	F8_DEVELOPER_MODE_ENABLED=1 F8_RESOURCE_DATABASE=1 F8_RESOURCE_UNIT_TEST=0 F8_LOG_LEVEL=$(F8_LOG_LEVEL) go test $(GO_TEST_VERBOSITY_FLAG) $(TEST_PACKAGES)

test-integration-benchmark: prebuild-check migrate-database $(SOURCES)
	$(call log-info,"Running benchmarks: $@")
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	F8_DEVELOPER_MODE_ENABLED=1 F8_RESOURCE_DATABASE=1 F8_RESOURCE_UNIT_TEST=0 F8_LOG_LEVEL=$(F8_LOG_LEVEL) go test -run=^$$ -bench=. -cpu 1,2,4 -test.benchmem $(GO_TEST_VERBOSITY_FLAG) $(TEST_PACKAGES) | grep -E "Bench|allocs"
