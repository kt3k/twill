#!/usr/bin/env bash
# Compares the Deno and Go CLIs on the sample project in project/ (scanner,
# extraction, minification). Builds the Go binary into out/, which is ignored.
set -euo pipefail
cd "$(dirname "$0")"
root=$PWD
mkdir -p out/cli
go build -C ../impl/go -o "$root/out/cli/twill-go" ./cmd/twill
status=0
run() {
  local label=$1; shift
  (cd ../impl/deno && deno run -A cli.ts --cwd "$root/project" --silent "$@") > "out/cli/$label.deno.css"
  out/cli/twill-go --cwd project --silent "$@" > "out/cli/$label.go.css"
  if ! diff -u "out/cli/$label.deno.css" "out/cli/$label.go.css" > "out/cli/$label.diff"; then
    echo "DIFF: $label (see conformance/out/cli/$label.diff)"; status=1
  else
    rm "out/cli/$label.diff"; echo "same: $label ($(wc -c < "out/cli/$label.deno.css") bytes)"
  fi
}
run build -i input.css
run minify -i input.css --minify
run default-input < /dev/null
exit $status
