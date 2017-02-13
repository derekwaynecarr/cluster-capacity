#! /bin/sh

# Assumptions:
# - cluster provisioned
# - KUBE_MASTER_API: master api url
# - KUBE_MASTER_API_PORT: master api port

KUBE_MASTER_API=${KUBE_MASTER_API:-http://localhost}
KUBE_MASTER_API_PORT=${KUBE_MASTER_API_PORT:-443}
KUBE_CONFIG=${KUBE_CONFIG:-~/.kube/config}

alias kubectl="kubectl --kubeconfig=${KUBE_CONFIG} --server=${KUBE_MASTER_API}:${KUBE_MASTER_API_PORT}"
#### pre-tests checks

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

function printError {
  echo -e "${RED}$1${NC}"
}

function printSuccess {
  echo -e "${GREEN}$1${NC}"
}

echo "####PRE-TEST CHECKS"
# check the cluster is available
kubectl version
if [ "$?" -ne 0 ]; then
  printError "Unable to connect to kubernetes cluster"
  exit 1
fi

# check the cluster-capacity namespace exists
kubectl get ns cluster-capacity
if [ "$?" -ne 0 ]; then
  kubectl create -f examples/namespace.yml
  if [ "$?" -ne 0 ]; then
    printError "Unable to create cluster-capacity namespace"
    exit 1
  fi
fi

echo ""
echo ""

#### TESTS
echo "####RUNNING TESTS"
echo ""
echo "# Running simple estimation of examples/pod.yaml"
./cluster-capacity --kubeconfig ${KUBE_CONFIG} --master ${KUBE_MASTER_API}:${KUBE_MASTER_API_PORT} --podspec=examples/pod.yaml | tee estimation.log
if [ -z "$(cat estimation.log | grep 'Termination reason')" ]; then
  printError "Missing termination reason"
  exit 1
fi

echo ""
echo "# Running simple estimation of examples/pod.yaml in verbose mode"
./cluster-capacity --kubeconfig ${KUBE_CONFIG} --master ${KUBE_MASTER_API}:${KUBE_MASTER_API_PORT} --podspec=examples/pod.yaml --verbose | tee estimation.log
if [ -z "$(cat estimation.log | grep 'Termination reason')" ]; then
  printError "Missing termination reason"
  exit 1
fi

echo ""
echo "Decrease resource in the cluster by running new pods"
kubectl create -f examples/rc.yml
if [ "$?" -ne 0 ]; then
  printError "Unable to create additional resources"
  exit 1
fi

while [ $(kubectl get pods | grep nginx | grep "Running" | wc -l) -ne 3 ]; do
  echo "waiting for pods to come up"
  sleep 1s
done

echo ""
echo "# Running simple estimation of examples/pod.yaml in verbose mode with less resources"
./cluster-capacity --kubeconfig ${KUBE_CONFIG} --master ${KUBE_MASTER_API}:${KUBE_MASTER_API_PORT} --podspec=examples/pod.yaml --verbose | tee estimation.log
if [ -z "$(cat estimation.log | grep 'Termination reason')" ]; then
  printError "Missing termination reason"
  exit 1
fi

echo ""
echo "Delete resource in the cluster by deleting rc"
kubectl delete -f examples/rc.yml

echo ""
echo ""
printSuccess "#### All tests passed"

exit 0