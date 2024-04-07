#!/bin/bash

# Copyright 2024 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit -o nounset -o xtrace

# Create a function to setup and run kind
function setup_kind {
  # Download and install kind
  curl -fsSL -o ./kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"
  install --mode=755 ./kind /usr/local/bin/kind

  # Create a kind cluster with a specific Kubernetes version that will match the hydrophone test
  cat <<EOF | kind create cluster --image kindest/node:${K8S_VERSION} --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF

  # Retrieve cluster information
  kubectl cluster-info --context kind-kind
  kubectl get nodes
}

# Setup and run hydrophone
function run_test {
  if [[ ! -f bin/hydrophone ]]; then
    echo "bin/hydrophone does not exist. Run make build to build it."
    exit 1
  fi

  bin/hydrophone \
    --output-dir ${ARTIFACTS}/results/ \
    --focus "${FOCUS}" \
    --skip "${SKIP}" \
    --namespace "${NAMESPACE}" \
    $EXTRA_ARGS | tee /tmp/test.log

  # Check if $CHECK_DURATION is set to true
  if [[ ${CHECK_DURATION} == "true" ]]; then
    # Check duration
    DURATION=$(grep -oP 'Ran \d+ of \d+ Specs in \K[0-9.]+(?= seconds)' /tmp/test.log | cut -d. -f1)

    if [[ ${DRYRUN} == "true" ]]; then
      if [[ ${DURATION} -gt ${DRYRUN_THRESHOLD} ]]; then
        echo "Focused test took too long to run. Expected less than ${DRYRUN_THRESHOLD} seconds, got ${DURATION} seconds"
        exit 1
      fi
    else
      if [[ ${DURATION} -lt ${DRYRUN_THRESHOLD} ]]; then
        echo "Focused test exited too quickly, check if dry-run is enabled. Expected more than ${DRYRUN_THRESHOLD} seconds, got ${DURATION} seconds"
        exit 1
      fi
    fi
  fi

  # If EXPECTED_NUM_TESTS is set, run the evaluate_test_num function
  if [[ ! -z ${EXPECTED_NUM_TESTS+x} ]]; then
    evaluate_test_num
  fi
}

function evaluate_test_num {
  NUM_TESTS=$(grep -oP 'Ran \K\d+(?= of \d+ Specs)' /tmp/test.log)

  # Check if NUM_TESTS is not equal to EXPECTED_NUM_TESTS
  if [[ ${NUM_TESTS} -ne ${EXPECTED_NUM_TESTS} ]]; then
    echo "Expected ${EXPECTED_NUM_TESTS} tests, got ${NUM_TESTS} tests"
    exit 1
  fi
}

# Default versions k8s and kind
K8S_VERSION=${K8S_VERSION:-v1.29.0}
KIND_VERSION=${KIND_VERSION:-v0.20.0}

# Maximum time (in seconds) for a dry run test
DRYRUN_THRESHOLD=${DRYRUN_DURATION:-5}

# Default variables for run
FOCUS=${FOCUS:-""}
SKIP=${SKIP:-""}
DRYRUN=${DRYRUN:-"false"}
NAMESPACE=${NAMESPACE:-""}
CONFORMANCE=${CONFORMANCE:-"false"}
EXTRA_ARGS=${EXTRA_ARGS:-""}
CHECK_DURATION=${CHECK_DURATION:-"false"}
SET_VERSION=${SET_VERSION:-"false"}

# Set the artifacts directory, defaulting to a local subdirectory
export ARTIFACTS="${ARTIFACTS:-${PWD}/_artifacts}"
mkdir -p "${ARTIFACTS}/results"

# if DRYRUN is set, add --dry-run to the EXTRA_ARGS
if [[ ${DRYRUN} == "true" ]]; then
  EXTRA_ARGS="${EXTRA_ARGS} --dry-run"
fi

# If CONFORMANCE is set, add --conformance to the EXTRA_ARGS
if [[ ${CONFORMANCE} == "true" ]]; then
  FOCUS="\\[Conformance\\]"
fi

# If SET_VERSION is set, set the K8S_VERSION to the value of SET_VERSION
if [[ ${SET_VERSION} == "true" ]]; then
  EXTRA_ARGS="${EXTRA_ARGS} --conformance-image registry.k8s.io/conformance:${K8S_VERSION}"
fi

setup_kind
run_test
