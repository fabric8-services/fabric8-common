#!/bin/bash

# Output command before executing
set -x

# Exit on error
set -e

# Source environment variables of the jenkins slave
# that might interest this worker.
# TODO(kwk): Fix this as it is broken if values of env vars contain spaces :/
# See https://github.com/openshiftio/openshiftio-cico-jobs/pull/186 for an approach
# using "declare -p" on the site that creates the .jenkins-env file.
function load_jenkins_vars() {
  if [ -e "jenkins-env.json" ]; then
    eval "$(./env-toolkit load -f jenkins-env.json \
              DEVSHIFT_TAG_LEN \
              QUAY_USERNAME \
              QUAY_PASSWORD \
              JENKINS_URL \
              GIT_BRANCH \
              GIT_COMMIT \
              BUILD_NUMBER \
              ghprbSourceBranch \
              ghprbActualCommit \
              BUILD_URL \
              ghprbPullId)"
  fi
}

function install_deps() {
  # We need to disable selinux for now, XXX
  /usr/sbin/setenforce 0 || :

  # Get all the deps in
  yum -y install --quiet \
    docker \
    make \
    git \
    curl

  service docker start

  echo 'CICO: Dependencies installed'
}

function cleanup_env {
  local exit_code=$?
  echo "CICO: Cleanup environment: Tear down test environment"
  make integration-test-env-tear-down
  echo "CICO: Exiting with ${exit_code}"
}

function prepare() {
  # Start "flow-heater" container to build in and run tests in.
  # Every make target that begins with "docker-" will be executed
  # in the resulting container.
  make docker-start
  make docker-check-go-format
  # Download Go dependencies
  make docker-deps
  # Check code for style violations (vet, etc).
  # make docker-analyze-go-code
  # Take Goa designs and generate code with it.
  make docker-generate
  # Build the wit and wit-cli binary
  make docker-build
  echo 'CICO: Preparation complete'
}

function run_tests_without_coverage() {
  make docker-test-unit-no-coverage
  make integration-test-env-prepare
  trap cleanup_env EXIT

  # # Check that postgresql container is healthy
  check_postgres_healthiness

  make docker-test-integration-no-coverage
  # make docker-test-remote-no-coverage
  echo "CICO: ran tests without coverage"
}

function run_go_benchmarks() {
  make integration-test-env-prepare
  trap cleanup_env EXIT

  # Check that postgresql container is healthy
  check_postgres_healthiness

  make docker-test-integration-benchmark
  echo "CICO: ran go benchmarks"
}

function check_postgres_healthiness(){
  echo "CICO: Waiting for postgresql container to be healthy...";
  while ! docker ps | grep postgres_integration_test | grep -q healthy; do
    printf .;
    sleep 1 ;
  done;
  echo "CICO: postgresql container is HEALTHY!";
}

function run_tests_with_coverage() {
  # Run the unit tests that generate coverage information
  make docker-test-unit
  make integration-test-env-prepare
  trap cleanup_env EXIT

  # # Check that postgresql container is healthy
  check_postgres_healthiness

  # # Run the integration tests that generate coverage information
  make docker-test-integration

  # # Run the remote tests that generate coverage information
  # make docker-test-remote

  # Output coverage
  make docker-coverage-all

  # Upload coverage to codecov.io
  cp tmp/coverage.mode* coverage.txt
  bash <(curl -s https://codecov.io/bash) -X search -f coverage.txt -t e0e851ea-3abc-44fe-86a4-3d3559e35e46 #-X fix

  echo "CICO: ran tests and uploaded coverage"
}

function cico_setup() {
  load_jenkins_vars;
  install_deps;
  prepare;
}
