#!/usr/bin/env bash

set -euo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
readonly OUTPUT="${1:-${ROOT_DIR}/_workspace/complexity/inventory.md}"
readonly GOCYCLO_VERSION="v0.6.0"
readonly GOCOGNIT_VERSION="v1.2.1"
readonly MAX_ALLOWED=9

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

tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

go run "github.com/fzipp/gocyclo/cmd/gocyclo@${GOCYCLO_VERSION}" "${go_files[@]}" >"${tmp_dir}/cyclo.txt"
go run "github.com/uudashr/gocognit/cmd/gocognit@${GOCOGNIT_VERSION}" -test "${go_files[@]}" >"${tmp_dir}/cognit.txt"

cyclo_status=0
go run "github.com/fzipp/gocyclo/cmd/gocyclo@${GOCYCLO_VERSION}" \
  -over "${MAX_ALLOWED}" "${go_files[@]}" >"${tmp_dir}/cyclo-violations.txt" || cyclo_status=$?
if (( cyclo_status != 0 && cyclo_status != 1 )); then
  exit "${cyclo_status}"
fi

cognit_status=0
go run "github.com/uudashr/gocognit/cmd/gocognit@${GOCOGNIT_VERSION}" \
  -over "${MAX_ALLOWED}" -test "${go_files[@]}" >"${tmp_dir}/cognit-violations.txt" || cognit_status=$?
if (( cognit_status != 0 && cognit_status != 1 )); then
  exit "${cognit_status}"
fi

awk -v root="${ROOT_DIR}/" '
FNR == NR {
  location=$NF
  sub(root, "", location)
  key=$3 "|" location
  cyclo[key]=$1
  package_name[key]=$2
  function_name[key]=$3
  source[key]=location
  keys[key]=1
  next
}
{
  location=$NF
  sub(root, "", location)
  key=$3 "|" location
  cognit[key]=$1
  package_name[key]=$2
  function_name[key]=$3
  source[key]=location
  keys[key]=1
}
END {
  for (key in keys) {
    c=cyclo[key]+0
    g=cognit[key]+0
    if (c > 9 || g > 9) {
      split(source[key], location_parts, ":")
      path=location_parts[1]
      line=location_parts[2]
      max_score=(c > g ? c : g)
      sum_score=c+g
      printf "%d|%d|%s|%s|%s|%s|%d|%d\n", max_score, sum_score, path, line, function_name[key], package_name[key], c, g
    }
  }
}' "${tmp_dir}/cyclo.txt" "${tmp_dir}/cognit.txt" \
  | sort -t '|' -k1,1nr -k2,2nr -k3,3 -k5,5 >"${tmp_dir}/violations.txt"

read -r cyclo_count cognit_count union_count max_cyclo max_cognit <<<"$(awk '
FNR == NR {
  location=$NF
  key=$3 "|" location
  cyclo[key]=$1
  keys[key]=1
  if ($1 > max_cyclo) max_cyclo=$1
  next
}
{
  location=$NF
  key=$3 "|" location
  cognit[key]=$1
  keys[key]=1
  if ($1 > max_cognit) max_cognit=$1
}
END {
  for (key in keys) {
    c=cyclo[key]+0
    g=cognit[key]+0
    if (c > 9) cyclo_count++
    if (g > 9) cognit_count++
    if (c > 9 || g > 9) union_count++
  }
  printf "%d %d %d %d %d", cyclo_count, cognit_count, union_count, max_cyclo, max_cognit
}' "${tmp_dir}/cyclo.txt" "${tmp_dir}/cognit.txt")"

threshold_cyclo_count="$(wc -l <"${tmp_dir}/cyclo-violations.txt" | tr -d ' ')"
threshold_cognit_count="$(wc -l <"${tmp_dir}/cognit-violations.txt" | tr -d ' ')"
if (( threshold_cyclo_count != cyclo_count || threshold_cognit_count != cognit_count )); then
  printf 'Threshold inventory mismatch: full=%d/%d over-%d=%d/%d\n' \
    "${cyclo_count}" "${cognit_count}" "${MAX_ALLOWED}" "${threshold_cyclo_count}" "${threshold_cognit_count}" >&2
  exit 1
fi

mkdir -p "$(dirname "${OUTPUT}")"
{
  printf '# Complexity inventory\n\n'
  printf 'Generated: %s  \n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'Base: `%s`  \n' "$(git -C "${ROOT_DIR}" rev-parse HEAD)"
  printf 'Tools: `gocyclo@%s`, `gocognit@%s`, threshold `-over %s`  \n\n' "${GOCYCLO_VERSION}" "${GOCOGNIT_VERSION}" "${MAX_ALLOWED}"
  printf '| Cyclomatic violations | Cognitive violations | Union | Max cycle | Max cognitive |\n'
  printf '|---:|---:|---:|---:|---:|\n'
  printf '| %d | %d | %d | %d | %d |\n\n' "${cyclo_count}" "${cognit_count}" "${union_count}" "${max_cyclo}" "${max_cognit}"
  printf 'Generated files are excluded only when their first 40 lines contain the canonical `// Code generated ... DO NOT EDIT.` marker.\n\n'
  printf '| Rank | File | Line | Function | Package | Cyclomatic | Cognitive |\n'
  printf '|---:|---|---:|---|---|---:|---:|\n'
  rank=0
  while IFS='|' read -r _max _sum path line function package cycle cognitive; do
    rank=$((rank + 1))
    printf '| %d | `%s` | %s | `%s` | `%s` | %s | %s |\n' "${rank}" "${path}" "${line}" "${function}" "${package}" "${cycle}" "${cognitive}"
  done <"${tmp_dir}/violations.txt"
} >"${OUTPUT}"

printf 'inventory=%s cyclomatic=%d cognitive=%d union=%d max=%d/%d\n' \
  "${OUTPUT}" "${cyclo_count}" "${cognit_count}" "${union_count}" "${max_cyclo}" "${max_cognit}"
