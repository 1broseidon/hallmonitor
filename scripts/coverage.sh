#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROFILE="${PROFILE:-coverage.out}"
COVERMODE="${COVERMODE:-atomic}"
PACKAGES="./..."
WITH_RACE=false
GENERATE_HTML=false

usage() {
  cat <<EOF
Usage: scripts/coverage.sh [options]

Options:
  -p, --packages <packages>   Go package pattern to test (default ./...)
      --race                  Enable -race when running go test
      --html                  Generate coverage.html alongside coverage.out
      --profile <file>        Override coverprofile output path (default coverage.out)
      --covermode <mode>      Override covermode (default atomic)
  -h, --help                  Show this help message
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--packages)
      PACKAGES="$2"
      shift 2
      ;;
    --race)
      WITH_RACE=true
      shift
      ;;
    --html)
      GENERATE_HTML=true
      shift
      ;;
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --covermode)
      COVERMODE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

CMD=(go test -covermode="${COVERMODE}" -coverprofile="${PROFILE}")
if [[ "${WITH_RACE}" == "true" ]]; then
  CMD+=(-race)
fi
CMD+=("${PACKAGES}")

echo "==> Running ${CMD[*]}"

pushd "${ROOT_DIR}" >/dev/null
"${CMD[@]}"

REPORT=$(go tool cover -func="${PROFILE}")
TOTAL_LINE=$(echo "${REPORT}" | tail -n 1)

echo "==> Package coverage (low to high)"
echo "${REPORT}" \
  | grep "github.com/1broseidon/hallmonitor/" \
  | awk '{path=$1; sub(/:[0-9]+:$/, "", path); sub("github.com/1broseidon/hallmonitor/", "", path); sub("/[^/]+$", "", path); if (path=="") path="root"; cov=$NF; gsub("%", "", cov); if (cov=="" || cov=="?" || cov=="-" ) cov=0; cov+=0; sum[path]+=cov; count[path]++} END {for (path in sum) if (count[path]>0) printf "%6.2f%%\t%s\n", sum[path]/count[path], path;}' \
  | sort -n

echo "==> ${TOTAL_LINE}"

if [[ "${GENERATE_HTML}" == "true" ]]; then
  HTML_OUT="${PROFILE%.out}.html"
  go tool cover -html="${PROFILE}" -o "${HTML_OUT}"
  echo "==> HTML report written to ${HTML_OUT}"
fi

popd >/dev/null

