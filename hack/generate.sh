#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# api_violations.report is required by openapi-gen tool.
# if this file is not present in target folder, code generation fails.
if [ ! -f "clientgo/openapi/api_violations.report" ]; then
  mkdir -p clientgo/openapi
  touch clientgo/openapi/api_violations.report
fi

export TERM="xterm-256color"

bold="$(tput bold)"
blue="$(tput setaf 4)"
normal="$(tput sgr0)"

function qualify-gvs() {
  APIS_PKG="$1"
  GROUPS_WITH_VERSIONS="$2"
  join_char=""
  res=""

  for GVs in ${GROUPS_WITH_VERSIONS}; do
    IFS=: read -r G Vs <<<"${GVs}"

    for V in ${Vs//,/ }; do
      res="$res$join_char$APIS_PKG/$G/$V"
      join_char=","
    done
  done

  echo "$res"
}

VGOPATH="$VGOPATH"
MODELS_SCHEMA="$MODELS_SCHEMA"
DEEPCOPY_GEN="$DEEPCOPY_GEN"
OPENAPI_GEN="$OPENAPI_GEN"
APPLYCONFIGURATION_GEN="$APPLYCONFIGURATION_GEN"
CLIENT_GEN="$CLIENT_GEN"
LISTER_GEN="$LISTER_GEN"
INFORMER_GEN="$INFORMER_GEN"

VIRTUAL_GOPATH="$(mktemp -d)"
trap 'rm -rf "$VIRTUAL_GOPATH"' EXIT

# Setup virtual GOPATH so the codegen tools work as expected.
(cd "$SCRIPT_DIR/.."; go mod download && "$VGOPATH" -o "$VIRTUAL_GOPATH")

export GOROOT="${GOROOT:-"$(go env GOROOT)"}"
export GOPATH="$VIRTUAL_GOPATH"
export GO111MODULE=off

CLIENT_GROUPS="ipam"
CLIENT_VERSION_GROUPS="ipam:v1alpha1"
ALL_VERSION_GROUPS="$CLIENT_VERSION_GROUPS"

echo "Generating ${blue}deepcopy${normal}"
"$DEEPCOPY_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input-dirs "$(qualify-gvs "github.com/onmetal/ipam/api" "$ALL_VERSION_GROUPS")" \
  -O zz_generated.deepcopy

echo "Generating ${blue}openapi${normal}"
"$OPENAPI_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input-dirs "$(qualify-gvs "github.com/onmetal/ipam/api" "$ALL_VERSION_GROUPS")" \
  --input-dirs "k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/version" \
  --input-dirs "k8s.io/api/core/v1" \
  --input-dirs "k8s.io/apimachinery/pkg/api/resource" \
  --output-package "github.com/onmetal/ipam/clientgo/openapi" \
  -O zz_generated.openapi \
  --report-filename "$SCRIPT_DIR/../clientgo/openapi/api_violations.report"

echo "Generating ${blue}applyconfiguration${normal}"
applyconfigurationgen_external_apis+=("k8s.io/apimachinery/pkg/apis/meta/v1")
applyconfigurationgen_external_apis+=("$(qualify-gvs "github.com/onmetal/ipam/api" "$ALL_VERSION_GROUPS")")
applyconfigurationgen_external_apis_csv=$(IFS=,; echo "${applyconfigurationgen_external_apis[*]}")
"$APPLYCONFIGURATION_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input-dirs "${applyconfigurationgen_external_apis_csv}" \
  --openapi-schema <("$MODELS_SCHEMA" --openapi-package "github.com/onmetal/ipam/clientgo/openapi" --openapi-title "ipam") \
  --output-package "github.com/onmetal/ipam/clientgo/applyconfiguration"

echo "Generating ${blue}client${normal}"
"$CLIENT_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input "$(qualify-gvs "github.com/onmetal/ipam/api" "$CLIENT_VERSION_GROUPS")" \
  --output-package "github.com/onmetal/ipam/clientgo" \
  --apply-configuration-package "github.com/onmetal/ipam/clientgo/applyconfiguration" \
  --clientset-name "ipam" \
  --input-base ""

echo "Generating ${blue}lister${normal}"
"$LISTER_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input-dirs "$(qualify-gvs "github.com/onmetal/ipam/api" "$CLIENT_VERSION_GROUPS")" \
  --output-package "github.com/onmetal/ipam/clientgo/listers"

echo "Generating ${blue}informer${normal}"
"$INFORMER_GEN" \
  --output-base "$GOPATH/src" \
  --go-header-file "$SCRIPT_DIR/boilerplate.go.txt" \
  --input-dirs "$(qualify-gvs "github.com/onmetal/ipam/api" "$CLIENT_VERSION_GROUPS")" \
  --versioned-clientset-package "github.com/onmetal/ipam/clientgo/ipam" \
  --listers-package "github.com/onmetal/ipam/clientgo/listers" \
  --output-package "github.com/onmetal/ipam/clientgo/informers" \
  --single-directory