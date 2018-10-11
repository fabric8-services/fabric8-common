#===============================================================================
# Testing has become a rather big and interconnected topic and that's why it
# has arrived in it's own file.
#
# We have to types of tests available:
#
#  1. unit tests
#  2. integration tests and
#  3. remote tests.
#
# While the unit tests can be executed fairly simply be running `go test`, the
# integration tests have a little bit more setup going on. The remote tests reqire
# availability of some remote servers such as Keycloak.
# That's why they are split up in three tests.
#
# Usage
# -----
# If you want to run the unit tests, type
#
#     $ make test-unit
#
# To run the integration tests, type
#
#     $ make test-integration
#
# To run the remote tests, type
#
#     $ make test-remote
#
# To run all tests, type
#
#     $ make test-all
#
# To output unit-test coverage profile information for each function, type
#
#     $ make coverage-unit
#
# To generate unit-test HTML representation of coverage profile (opens a browser), type
#
#     $ make coverage-unit-html
#
# If you replace the "unit" with "integration" or "remote" you get the same for integration
# or remote tests.
#
# To output all coverage profile information for each function, type
#
#     $ make coverage-all
#
# Artifacts and coverage modes
# ----------------------------
# Each package generates coverage outputs under tmp/coverage/$(PACKAGE) where
# $(PACKAGE) resolves to the Go package. Here's an example of a coverage file
# for the package "github.com/fabric8-services/fabric8-wit/models" with coverage mode
# "set" generated by the unit tests:
#
#   tmp/coverage/github.com/fabric8-services/fabric8-wit/models/coverage.unit.mode-set
#
# For unit-tests all results are combined into this file:
#
#   tmp/coverage.unit.mode-$(COVERAGE_MODE)
#
# For integration-tests all results are combined into this file:
#
#   tmp/coverage.integration.mode-$(COVERAGE_MODE)
#
# For remote-tests all results are combined into this file:
#
#   tmp/coverage.remote.mode-$(COVERAGE_MODE)
#
# The overall coverage gets combined into this file:
#
#   tmp/coverage.mode-$(COVERAGE_MODE)
#
# The $(COVERAGE_MODE) in each filename indicates what coverage mode was used.
#
# These are possible coverage modes (see https://blog.golang.org/cover):
#
# 	set: did each statement run? (default)
# 	count: how many times did each statement run?
# 	atomic: like count, but counts precisely in parallel programs
#
# To choose another coverage mode, simply prefix the invovation of `make`:
#
#     $ COVERAGE_MODE=count make test-unit
#===============================================================================

# mode can be: set, count, or atomic
COVERAGE_MODE ?= set

# By default no go test calls will use the -v switch when running tests.
# But if you want you can enable that by setting GO_TEST_VERBOSITY_FLAG=-v
GO_TEST_VERBOSITY_FLAG ?= 

# By default reduce the amount of log output from tests
F8_LOG_LEVEL ?= error

# Output directory for coverage information
COV_DIR = $(TMP_PATH)/coverage

# Files that combine package coverages for unit- and integration-tests separately
COV_PATH_UNIT = $(TMP_PATH)/coverage.unit.mode-$(COVERAGE_MODE)

# File that stores overall coverge for all packages and unit- integration- and remote-tests
COV_PATH_OVERALL = $(TMP_PATH)/coverage.mode-$(COVERAGE_MODE)

# This pattern excludes some folders from the coverage calculation (see grep -v)
ALL_PKGS_EXCLUDE_PATTERN = "vendor\|account\/tenant\|app\'\|tool\/cli\|design\|client\|test"

# This pattern excludes some folders from the go code analysis
GOANALYSIS_PKGS_EXCLUDE_PATTERN="vendor|account/tenant|app|client|tool/cli"
GOANALYSIS_DIRS=$(shell go list -f {{.Dir}} ./... | grep -v -E $(GOANALYSIS_PKGS_EXCLUDE_PATTERN))

# docker-compose for integration tests
DOCKER_COMPOSE_BIN := $(shell command -v $(DOCKER_COMPOSE_BIN_NAME) 2> /dev/null)
DOCKER_COMPOSE_FILE = $(CUR_DIR)/.make/docker-compose.integration-test.yaml
# Alternative path to docker-compose (if downloaded)
DOCKER_COMPOSE_BIN_ALT = $(TMP_PATH)/docker-compose

#-------------------------------------------------------------------------------
# Normal test targets
#
# These test targets are the ones that will be invoked from the outside. If
# they are called and the artifacts already exist, then the artifacts will
# first be cleaned and recreated. This ensures that the tests are always
# executed.
#-------------------------------------------------------------------------------

.PHONY: test-all
## Runs test-unit and test-integration targets.
test-all: prebuild-check test-unit

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

#-------------------------------------------------------------------------------
# Inspect coverage of unit tests, integration or remote tests in either pure
# console mode or in a browser (*-html).
#
# If the test coverage files to be evaluated already exist, then no new
# tests are executed. If they don't exist, we first run the tests.
#-------------------------------------------------------------------------------

# Prints the total coverage of a given package.
# The total coverage is printed as the last argument in the
# output of "go tool cover". If the requested test name (first argument)
# Is *, then unit, integration and remote tests will be combined
define print-package-coverage
$(eval TEST_NAME:=$(1))
$(eval PACKAGE_NAME:=$(2))
$(eval COV_FILE:="$(COV_DIR)/$(PACKAGE_NAME)/coverage.$(TEST_NAME).mode-$(COVERAGE_MODE)")
 @if [ "$(TEST_NAME)" == "*" ]; then \
  UNIT_FILE=`echo $(COV_FILE) | sed 's/*/unit/'`; \
  INTEGRATON_FILE=`echo $(COV_FILE) | sed 's/*/integration/'`; \
  REMOTE_FILE=`echo $(COV_FILE) | sed 's/*/remote/'`; \
 	COV_FILE=`echo $(COV_FILE) | sed 's/*/combined/'`; \
	if [ ! -e $${UNIT_FILE} ]; then \
		if [ ! -e $${INTEGRATION_FILE} ]; then \
			COV_FILE=$${REMOTE_FILE}; \
		else \
			COV_FILE=$${INTEGRATION_FILE}; \
		fi; \
	else \
		if [ ! -e $${INTEGRATION_FILE} ]; then \
			COV_FILE=$${UNIT_FILE}; \
		else \
			$(GOCOVMERGE_BIN) $${UNIT_FILE} $${INTEGRATION_FILE} $${REMOTE_FILE} > $${COV_FILE}; \
		fi; \
	fi; \
else \
	COV_FILE=$(COV_FILE); \
fi; \
if [ -e "$${COV_FILE}" ]; then \
	VAL=`go tool cover -func=$${COV_FILE} \
		| grep '^total:' \
		| grep '\S\+$$' -o \
		| sed 's/%//'`; \
	printf "%-80s %#5.2f%%\n" "$(PACKAGE_NAME)" "$${VAL}"; \
else \
	printf "%-80s %6s\n" "$(PACKAGE_NAME)" "n/a"; \
fi
endef

$(COV_PATH_OVERALL): $(GOCOVMERGE_BIN)
	@echo "generating '$(COV_PATH_OVERALL)'..."
	@$(GOCOVMERGE_BIN) $(COV_PATH_UNIT) > $(COV_PATH_OVERALL)

# Iterates over every package and prints its test coverage
# for a given test name ("unit", "integration" or "remote").
define package-coverage
$(eval TEST_NAME:=$(1))
@printf "\n\nPackage coverage:\n"
$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
$(foreach package, $(TEST_PACKAGES), $(call print-package-coverage,$(TEST_NAME),$(package)))
endef

# Console coverage output:

# First parameter: file to do in-place replacement with.
# Delete the lines containing /bindata_assetfs.go
# define cleanup-coverage-file
# @echo "cleanup-coverage-file: '$(1)' (COV_PATH_UNIT=$(COV_PATH_UNIT))"
# @sed -i '/.*\/bindata_assetfs\.go.*/d' $(1)
# @sed -i '/.*\/sqlbindata\.go.*/d' $(1)
# @echo "cleanup-coverage-file done"
# endef

.PHONY: coverage-unit
## Output coverage profile information for each function (only based on unit-tests).
## Re-runs unit-tests if coverage information is outdated.
coverage-unit: prebuild-check $(COV_PATH_UNIT)
	# $(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@go tool cover -func=$(COV_PATH_UNIT)
	$(call package-coverage,unit)

.PHONY: coverage-all
## Output coverage profile information for each function.
## Re-runs unit-, integration- and remote-tests if coverage information is outdated.
coverage-all: prebuild-check clean-coverage-overall $(COV_PATH_OVERALL)
	@echo "running coverage all..."
	# $(call cleanup-coverage-file,$(COV_PATH_OVERALL))
	@go tool cover -func=$(COV_PATH_OVERALL)
	$(call package-coverage,*)

# HTML coverage output:

.PHONY: coverage-unit-html
## Generate HTML representation (and show in browser) of coverage profile (based on unit tests).
## Re-runs unit tests if coverage information is outdated.
coverage-unit-html: prebuild-check $(COV_PATH_UNIT)
	# $(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@go tool cover -html=$(COV_PATH_UNIT)

.PHONY: coverage-all-html
## Output coverage profile information for each function.
## Re-runs unit-, integration- and remote-tests if coverage information is outdated.
coverage-all-html: prebuild-check clean-coverage-overall $(COV_PATH_OVERALL)
	# $(call cleanup-coverage-file,$(COV_PATH_OVERALL))
	@go tool cover -html=$(COV_PATH_OVERALL)

# Experimental:

.PHONY: gocov-unit-annotate
## (EXPERIMENTAL) Show actual code and how it is covered with unit tests.
##                This target only runs the tests if the coverage file does exist.
gocov-unit-annotate: prebuild-check $(GOCOV_BIN) $(COV_PATH_UNIT)
	# $(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@$(GOCOV_BIN) convert $(COV_PATH_UNIT) | $(GOCOV_BIN) annotate -

.PHONY: .gocov-unit-report
.gocov-unit-report: prebuild-check $(GOCOV_BIN) $(COV_PATH_UNIT)
	# $(call cleanup-coverage-file,$(COV_PATH_UNIT))
	@$(GOCOV_BIN) convert $(COV_PATH_UNIT) | $(GOCOV_BIN) report

#-------------------------------------------------------------------------------
# Test artifacts are coverage files for unit, integration and remote tests.
#-------------------------------------------------------------------------------

# The test-package function executes tests for a package and saves the collected
# coverage output to a directory. After storing the coverage information it is
# also appended to a file of choice (without the "mode"-line)
#
# Parameters:
#  1. Test name (e.g. "unit", "integration" or "remote")
#  2. package name "github.com/fabric8-services/fabric8-wit/model"
#  3. File in which to combine the output
#  4. Path to file in which to store names of packages that failed testing
#  5. Environment variable (in the form VAR=VALUE) to be specified for running
#     the test. For multiple environment variables, pass "VAR1=VAL1 VAR2=VAL2".
define test-package
$(eval TEST_NAME := $(1))
$(eval PACKAGE_NAME := $(2))
$(eval COMBINED_OUT_FILE := $(3))
$(eval ERRORS_FILE := $(4))
$(eval ENV_VAR := $(5))
$(eval ALL_PKGS_COMMA_SEPARATED := $(6))
@mkdir -p $(COV_DIR)/$(PACKAGE_NAME);
$(eval COV_OUT_FILE := $(COV_DIR)/$(PACKAGE_NAME)/coverage.$(TEST_NAME).mode-$(COVERAGE_MODE))
@$(ENV_VAR) F8_DEVELOPER_MODE_ENABLED=1 F8_POSTGRES_HOST=$(F8_POSTGRES_HOST) F8_LOG_LEVEL=$(F8_LOG_LEVEL) \
	go test  $(PACKAGE_NAME) \
		$(GO_TEST_VERBOSITY_FLAG) \
		-coverprofile $(COV_OUT_FILE) \
		-coverpkg $(ALL_PKGS_COMMA_SEPARATED) \
		-covermode=$(COVERAGE_MODE) \
		-timeout 10m \
		$(EXTRA_TEST_PARAMS) \
	|| echo $(PACKAGE_NAME) >> $(ERRORS_FILE)

@if [ -e "$(COV_OUT_FILE)" ]; then \
	if [ ! -e "$(COMBINED_OUT_FILE)" ]; then \
		cp $(COV_OUT_FILE) $(COMBINED_OUT_FILE); \
	else \
		cp $(COMBINED_OUT_FILE) $(COMBINED_OUT_FILE).tmp; \
		$(GOCOVMERGE_BIN) $(COV_OUT_FILE) $(COMBINED_OUT_FILE).tmp > $(COMBINED_OUT_FILE); \
	fi \
fi
endef

# Exits the makefile with an error if the file (first parameter) exists.
# Before exiting, the contents of the passed file is printed.
define check-test-results
$(eval ERRORS_FILE := $(1))
@if [ -e "$(ERRORS_FILE)" ]; then \
echo ""; \
echo "ERROR: The following packages did not pass the tests:"; \
echo "-----------------------------------------------------"; \
cat $(ERRORS_FILE); \
echo "-----------------------------------------------------"; \
echo ""; \
exit 1; \
fi
endef

# NOTE: We don't have prebuild-check as a dependency here because it would cause
#       the recipe to be always executed.
$(COV_PATH_UNIT): $(SOURCES) $(GOCOVMERGE_BIN)
	$(eval TEST_NAME := unit)
	$(eval ERRORS_FILE := $(TMP_PATH)/errors.$(TEST_NAME))
	$(call log-info,"Running test: $(TEST_NAME)")
	@mkdir -p $(COV_DIR)
	@echo "mode: $(COVERAGE_MODE)" > $(COV_PATH_UNIT)
	@-rm -f $(ERRORS_FILE)
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	$(eval ALL_PKGS_COMMA_SEPARATED:=$(shell echo $(TEST_PACKAGES)  | tr ' ' ,))
	$(foreach package, $(TEST_PACKAGES), $(call test-package,$(TEST_NAME),$(package),$(COV_PATH_UNIT),$(ERRORS_FILE),,$(ALL_PKGS_COMMA_SEPARATED)))
	$(call check-test-results,$(ERRORS_FILE))

#-------------------------------------------------------------------------------
# Additional tools to build
#-------------------------------------------------------------------------------

$(GOCOV_BIN): prebuild-check
	@cd $(VENDOR_DIR)/github.com/axw/gocov/gocov/ && go build

$(GOCOVMERGE_BIN): prebuild-check
	@cd $(VENDOR_DIR)/github.com/wadey/gocovmerge && go build

#-------------------------------------------------------------------------------
# Clean targets
#-------------------------------------------------------------------------------

CLEAN_TARGETS += clean-coverage
.PHONY: clean-coverage
## Removes all coverage files
clean-coverage: clean-coverage-unit clean-coverage-overall
	-@rm -rf $(COV_DIR)

CLEAN_TARGETS += clean-coverage-overall
.PHONY: clean-coverage-overall
## Removes overall coverage file
clean-coverage-overall:
	-@rm -f $(COV_PATH_OVERALL)

CLEAN_TARGETS += clean-coverage-unit
.PHONY: clean-coverage-unit
## Removes unit test coverage file
clean-coverage-unit:
	-@rm -f $(COV_PATH_UNIT)

# Downloads docker-compose to tmp/docker-compose if it does not already exist.
define download-docker-compose
	@if [ ! -f "$(DOCKER_COMPOSE_BIN_ALT)" ]; then \
		echo "Downloading docker-compose to $(DOCKER_COMPOSE_BIN_ALT)"; \
		UNAME_S=$(shell uname -s); \
		UNAME_M=$(shell uname -m); \
		URL="https://github.com/docker/compose/releases/download/1.11.2/docker-compose-$${UNAME_S}-$${UNAME_M}"; \
		curl --silent -L $${URL} > $(DOCKER_COMPOSE_BIN_ALT); \
		chmod +x $(DOCKER_COMPOSE_BIN_ALT); \
	fi
endef

.PHONY: integration-test-env-prepare
integration-test-env-prepare:
ifdef DOCKER_COMPOSE_BIN
	@$(DOCKER_COMPOSE_BIN) -f $(DOCKER_COMPOSE_FILE) up -d
else
ifneq ($(OS),Windows_NT)
	$(call download-docker-compose)
	@$(DOCKER_COMPOSE_BIN_ALT) -f $(DOCKER_COMPOSE_FILE) up -d
else
	$(error The "$(DOCKER_COMPOSE_BIN_NAME)" executable could not be found in your PATH)
endif
endif

.PHONY: integration-test-env-tear-down
integration-test-env-tear-down:
ifdef DOCKER_COMPOSE_BIN
	@$(DOCKER_COMPOSE_BIN) -f $(DOCKER_COMPOSE_FILE) down
else
ifneq ($(OS),Windows_NT)
	$(call download-docker-compose)
	@$(DOCKER_COMPOSE_BIN_ALT) -f $(DOCKER_COMPOSE_FILE) down
else
	$(error The "$(DOCKER_COMPOSE_BIN_NAME)" executable could not be found in your PATH)
endif
endif

.PHONY: test-integration-no-coverage
test-integration-no-coverage: prebuild-check $(SOURCES)
	$(call log-info,"Running test: $@")
	$(eval TEST_PACKAGES:=$(shell go list ./... | grep -v $(ALL_PKGS_EXCLUDE_PATTERN)))
	F8_DEVELOPER_MODE_ENABLED=1 F8_RESOURCE_DATABASE=1 F8_RESOURCE_UNIT_TEST=0 go test $(GO_TEST_VERBOSITY_FLAG) $(TEST_PACKAGES)
