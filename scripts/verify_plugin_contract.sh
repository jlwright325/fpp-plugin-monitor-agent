#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$ROOT_DIR/.." && pwd)"
FP_JSON="$REPO_ROOT/docs/contract-fingerprints.json"

if ! command -v jq >/dev/null 2>&1; then
  echo "verify_plugin_contract.sh: jq is required" >&2
  exit 1
fi

if [[ ! -f "$FP_JSON" ]]; then
  echo "Missing $FP_JSON" >&2
  exit 1
fi

failures=0

while IFS= read -r file; do
  rel="$REPO_ROOT/$file"
  if [[ ! -f "$rel" ]]; then
    echo "FAIL: expected file missing: $file" >&2
    failures=$((failures + 1))
    continue
  fi
  while IFS= read -r needle; do
    if ! grep -Fq -- "$needle" "$rel"; then
      echo "FAIL: $file does not contain required substring (contract drift):" >&2
      echo "  → $needle" >&2
      failures=$((failures + 1))
    fi
  done < <(jq -r --arg f "$file" '.fileAssertions[] | select(.file == $f) | .mustContain[]' "$FP_JSON")
done < <(jq -r '.fileAssertions[].file' "$FP_JSON" | sort -u)

ver="$(jq -r '.contractVersion' "$FP_JSON")"
echo "OK: plugin contract fingerprints satisfied (contractVersion=$ver)"

if [[ "$failures" -ne 0 ]]; then
  echo "verify_plugin_contract.sh: $failures check(s) failed — update docs/PLUGIN_API_CONTRACT.md and docs/contract-fingerprints.json together." >&2
  exit 1
fi
