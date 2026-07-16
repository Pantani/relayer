#!/usr/bin/env bash

set -u

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly MAX_ALLOWED=9
readonly GOCYCLO_VERSION="${GOCYCLO_VERSION:-v0.6.0}"
readonly GOCOGNIT_VERSION="${GOCOGNIT_VERSION:-v1.2.1}"

cyclomatic_status=0
cognitive_status=0
go_files=()

while IFS= read -r file; do
  if ! sed -n '1,40p' "${ROOT_DIR}/${file}" | grep -Eq '^// Code generated .* DO NOT EDIT[.]$'; then
    go_files+=("${ROOT_DIR}/${file}")
  fi
done < <(git -C "${ROOT_DIR}" ls-files --cached --others --exclude-standard -- '*.go')

if (( ${#go_files[@]} == 0 )); then
  echo "No handwritten Go files found." >&2
  exit 1
fi

echo "Cyclomatic complexity (maximum allowed: ${MAX_ALLOWED})"
go run "github.com/fzipp/gocyclo/cmd/gocyclo@${GOCYCLO_VERSION}" \
  -over "${MAX_ALLOWED}" \
  "${go_files[@]}" || cyclomatic_status=$?

echo
echo "Cognitive complexity (maximum allowed: ${MAX_ALLOWED})"
go run "github.com/uudashr/gocognit/cmd/gocognit@${GOCOGNIT_VERSION}" \
  -over "${MAX_ALLOWED}" \
  -test \
  "${go_files[@]}" || cognitive_status=$?

if (( cyclomatic_status != 0 || cognitive_status != 0 )); then
  echo
  echo "Complexity gate failed. Every handwritten Go function must score below 10 in both metrics." >&2
  exit 1
fi

echo
echo "Complexity gate passed. Every handwritten Go function scores below 10 in both metrics."
